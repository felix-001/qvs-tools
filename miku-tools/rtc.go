package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/pion/dtls/v2/pkg/crypto/selfsign"
	"github.com/pion/stun"
)

const (
	serverAddr = "127.0.0.1:9008"
	stunMagic  = 0x2112A442
)

func (s *Parser) rtcMemLeakTest() {
	sessionId := s.startRtcPlay()
	// 1. 创建UDP连接
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 2. 发送第一个STUN Binding Request (Frame 1)
	fmt.Println("Sending STUN Binding Request...")
	sendStunBindingRequest(conn, "374b612f724a6a684c727735", sessionId, false)

	// 3. 接收STUN Binding Response (Frame 2)
	fmt.Println("Waiting for STUN Binding Response...")
	recvStunResponse(conn)

	// 4. 建立DTLS连接 (Frame 3-6)
	fmt.Println("Starting DTLS handshake...")
	dtlsConn := establishDTLS(conn)
	defer dtlsConn.Close()

	// 5. 发送第二个STUN Binding Request (Frame 7)
	fmt.Println("Sending second STUN Binding Request with USE-CANDIDATE...")
	sendStunBindingRequest(conn, "653576723347794147344471", sessionId, true)

	var rtpPacketCount int
	buf := make([]byte, 1500) // 假设最大包大小为1500字节
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("过去5秒内读到的RTP包个数: %d\n", rtpPacketCount)
			rtpPacketCount = 0
		default:
			// 非阻塞读取
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			_, err := conn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// 超时错误，继续下一次循环
					continue
				}
				// 其他错误，退出协程
				fmt.Printf("读取数据出错: %v\n", err)
				return
			}
			// 简单假设每个读取到的数据都是RTP包
			rtpPacketCount++
		}
	}
}

// 发送STUN Binding请求
func sendStunBindingRequest(conn net.Conn, transactionID, sessionId string, useCandidate bool) {
	// 创建STUN消息
	msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	// 设置事务ID
	txID, _ := hex.DecodeString(transactionID)
	copy(msg.TransactionID[:], txID)

	// 添加USERNAME属性
	username := stun.NewUsername(sessionId)
	username.AddTo(msg)

	// 添加GOOG-NETWORK-INFO属性 (0xC057)
	googleInfo := []byte{0x00, 0x00, 0x03, 0xE7} // ID:0, Cost:999
	// 由于 stun.AttrGoogleNetworkInfo 未定义，手动定义属性类型
	const customGoogleNetworkInfoAttr stun.AttrType = 0xC057
	msg.Add(customGoogleNetworkInfoAttr, googleInfo)

	// 添加ICE-CONTROLLING属性
	tieBreaker, _ := hex.DecodeString("ffefdae779460647")
	iceControlling := make([]byte, 8)
	copy(iceControlling, tieBreaker)
	msg.Add(stun.AttrICEControlling, iceControlling)

	// 添加PRIORITY属性
	if useCandidate {
		priority := make([]byte, 4)
		binary.BigEndian.PutUint32(priority, 1845501695)
		msg.Add(stun.AttrPriority, priority)
	}

	// 添加USE-CANDIDATE属性 (Frame 7)
	if useCandidate {
		msg.Add(stun.AttrUseCandidate, []byte{})
	}

	// 计算并添加MESSAGE-INTEGRITY
	// 注意：实际应用中需要真实密钥，这里使用抓包中的硬编码值
	var integrity []byte
	if useCandidate {
		// Frame 7的HMAC-SHA1
		integrity, _ = hex.DecodeString("b5acd510cd7affb2de82901b1356f5c5f8b2b8eb")
	} else {
		// 占位20字节 (Frame 1)
		integrity = make([]byte, 20)
	}
	msg.Add(stun.AttrMessageIntegrity, integrity)

	// 计算并添加FINGERPRINT
	// 实际应使用CRC32计算，这里使用抓包值
	var fingerprint []byte
	if useCandidate {
		// Frame 7的CRC32
		fp, _ := hex.DecodeString("d19c165b")
		fingerprint = fp
	} else {
		// 占位4字节 (Frame 1)
		fingerprint = make([]byte, 4)
	}
	msg.Add(stun.AttrFingerprint, fingerprint)

	// 发送STUN消息
	conn.Write(msg.Raw)
}

// 接收STUN响应
func recvStunResponse(conn net.Conn) {
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println(err)
	}

	// 解析STUN响应
	var msg stun.Message
	if err := stun.Decode(buf[:n], &msg); err != nil {
		log.Println(err)
	}

	// 打印响应信息
	if msg.Type == stun.BindingSuccess {
		fmt.Println("Received STUN Binding Success Response")
		// 解析XOR-MAPPED-ADDRESS
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(&msg); err == nil {
			fmt.Printf("XOR-MAPPED-ADDRESS: %s:%d\n", xorAddr.IP, xorAddr.Port)
		}
	}
}

// 建立DTLS连接
func establishDTLS(udpConn net.Conn) *dtls.Conn {
	// 生成自签名证书
	cert, err := selfsign.GenerateSelfSigned()
	if err != nil {
		panic(err)
	}

	// 配置DTLS客户端
	config := &dtls.Config{
		Certificates:         []tls.Certificate{cert},
		InsecureSkipVerify:   true, // 跳过证书验证
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
		CipherSuites: []dtls.CipherSuiteID{
			dtls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}

	// 建立DTLS连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dtlsConn, err := dtls.ClientWithContext(ctx, udpConn, config)
	if err != nil {
		panic(err)
	}

	fmt.Println("DTLS handshake completed successfully")
	return dtlsConn
}

type RtcPlayResp struct {
	Code      int    `json:"code"`
	Server    int    `json:"server"`
	SDP       string `json:"sdp"`
	SessionID string `json:"sessionid"`
}

func (s *Parser) startRtcPlay() string {
	clientIp := ClientInfo{
		OSName:         "Mac OS",
		OSVersion:      "10.15.7",
		BrowserName:    "Chrome",
		BrowserVersion: "136.0.0.0",
		SDKVersion:     "1.1.1-alpha.2",
	}

	addr := fmt.Sprintf("webrtc://127.0.0.1:2985/live/teststream")
	editPrompt := EditPrompt{
		StreamURL: addr,
		ClientIP:  clientIp,
		SDP:       sdp,
	}
	jsonData, err := json.Marshal(editPrompt)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return ""
	}
	url := "http://127.0.0.1:2985/rtc/v1/play"
	// 创建一个 HTTP POST 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("创建请求出错: %v\n", err)
		return ""
	}

	// 设置请求头，指定 Content-Type 为 application/json
	req.Header.Set("Content-Type", "application/json")

	// 创建 HTTP 客户端并发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("发送请求出错: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体出错: %v\n", err)
		return ""
	}

	// 打印响应状态码和响应体
	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应体: %s\n", string(body))
	// 解析响应体为结构体
	var rtcPlayResp RtcPlayResp
	if err := json.Unmarshal(body, &rtcPlayResp); err != nil {
		fmt.Printf("解析响应体出错: %v\n", err)
		return ""
	}
	return rtcPlayResp.SessionID
}

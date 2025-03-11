package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func (p *Parser) Gb() {

	// 服务器地址
	serverAddr := fmt.Sprintf("%s:55068", p.conf.Ip)

	// 连接到服务器
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接到服务器失败: %v\n", err)
		return
	}
	defer conn.Close()

	// 要发送的数据包
	data := make([]byte, 1000000000) //[]byte("hello world, 111111111111111111111111111111111111111111111111111111")

	// 发送数据包
	for i := 0; i < 10000000; i++ {
		_, err = conn.Write(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "发送数据失败: %v\n", err)
			return
		}

		fmt.Println("数据发送成功", i)
		//time.Sleep(1 * time.Second)
	}
	/*
		for {
			// 读取服务器返回的数据
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Fprintf(os.Stderr, "读取数据失败: %v\n", err)
				return
			}

			fmt.Printf("服务器返回: %s\n", string(buf[:n]))
		}
	*/
}

func (p *Parser) GbCli() {
	// SIP服务器地址
	sipServer := fmt.Sprintf("%s:5061", p.conf.Ip)

	// 设备信息
	deviceID := "34020000001320000001"
	realm := "3402000000"
	password := "123456"

	// 连接到SIP服务器
	conn, err := net.Dial("tcp", sipServer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接到SIP服务器失败: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("连接到SIP服务器成功")

	// 发送REGISTER请求
	callID := generateCallID()
	nonce := ""

	// 第一次注册请求（不带认证信息）
	registerReq := createRegisterRequest(deviceID, p.conf.Ip, callID, "", "", "")
	_, err = conn.Write([]byte(registerReq))
	if err != nil {
		fmt.Fprintf(os.Stderr, "发送注册请求失败: %v\n", err)
		return
	}

	// 读取401响应获取nonce
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取响应失败: %v\n", err)
		return
	}
	response := string(buf[:n])
	nonce = extractNonce(response)

	// 计算认证信息
	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", deviceID, realm, password))
	ha2 := md5Hash(fmt.Sprintf("REGISTER:sip:%s@%s", deviceID, p.conf.Ip))
	response = md5Hash(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	// 发送带认证信息的REGISTER请求
	authenticatedRegister := createRegisterRequest(deviceID, p.conf.Ip, callID, realm, nonce, response)
	_, err = conn.Write([]byte(authenticatedRegister))
	if err != nil {
		fmt.Fprintf(os.Stderr, "发送认证注册请求失败: %v\n", err)
		return
	}
	fmt.Println("注册成功")

	// 启动保活协程
	go func() {
		for {
			time.Sleep(60 * time.Second)
			keepaliveReq := createKeepAliveRequest(deviceID, p.conf.Ip)
			_, err := conn.Write([]byte(keepaliveReq))
			if err != nil {
				fmt.Fprintf(os.Stderr, "发送保活请求失败: %v\n", err)
				return
			}
		}
	}()

	// 主循环处理接收到的消息
	for {
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取消息失败: %v\n", err)
			return
		}

		message := string(buf[:n])
		if strings.Contains(message, "MESSAGE") && strings.Contains(message, "Catalog") {
			fmt.Println("收到Catalog请求")
			// 处理Catalog请求
			catalogResponse := createCatalogResponse(deviceID, p.conf.Ip, extractCallID(message))
			_, err = conn.Write([]byte(catalogResponse))
			if err != nil {
				fmt.Fprintf(os.Stderr, "发送Catalog响应失败: %v\n", err)
				return
			}
		}
	}
}

// 辅助函数
func generateCallID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func md5Hash(text string) string {
	hash := md5.New()
	hash.Write([]byte(text))
	return hex.EncodeToString(hash.Sum(nil))
}

func extractNonce(response string) string {
	// 简单实现，实际应该使用正则表达式
	if i := strings.Index(response, "nonce="); i >= 0 {
		return response[i+7 : i+39] // 假设nonce长度为32
	}
	return ""
}

func extractCallID(message string) string {
	// 简单实现，实际应该使用正则表达式
	if i := strings.Index(message, "Call-ID: "); i >= 0 {
		end := strings.Index(message[i:], "\r\n")
		return message[i+9 : i+end]
	}
	return ""
}

func createRegisterRequest(deviceID, serverIP, callID, realm, nonce, response string) string {
	var auth string
	if nonce != "" {
		auth = fmt.Sprintf(`Authorization: Digest username="%s", realm="%s", nonce="%s", uri="sip:%s@%s", response="%s", algorithm=MD5\r\n`,
			deviceID, realm, nonce, deviceID, serverIP, response)
	}

	return fmt.Sprintf("REGISTER sip:%s@%s SIP/2.0\r\n"+
		"Via: SIP/2.0/TCP %s:5060;branch=z9hG4bK%s\r\n"+
		"From: <sip:%s@%s>;tag=%s\r\n"+
		"To: <sip:%s@%s>\r\n"+
		"Call-ID: %s\r\n"+
		"CSeq: 1 REGISTER\r\n"+
		"Contact: <sip:%s@%s:5060>\r\n"+
		"%s"+
		"Max-Forwards: 70\r\n"+
		"User-Agent: IPC\r\n"+
		"Expires: 3600\r\n"+
		"Content-Length: 0\r\n\r\n",
		deviceID, serverIP,
		serverIP, generateCallID(),
		deviceID, serverIP, generateCallID(),
		deviceID, serverIP,
		callID,
		deviceID, serverIP,
		auth)
}

func createKeepAliveRequest(deviceID, serverIP string) string {
	return fmt.Sprintf("MESSAGE sip:%s@%s SIP/2.0\r\n"+
		"Via: SIP/2.0/TCP %s:5060;branch=z9hG4bK%s\r\n"+
		"From: <sip:%s@%s>;tag=%s\r\n"+
		"To: <sip:%s@%s>\r\n"+
		"Call-ID: %s\r\n"+
		"CSeq: 1 MESSAGE\r\n"+
		"Content-Type: Application/MANSCDP+xml\r\n"+
		"Max-Forwards: 70\r\n"+
		"User-Agent: IPC\r\n"+
		"Content-Length: 0\r\n\r\n",
		deviceID, serverIP,
		serverIP, generateCallID(),
		deviceID, serverIP, generateCallID(),
		deviceID, serverIP,
		generateCallID())
}

func createCatalogResponse(deviceID, serverIP, callID string) string {
	body := `<?xml version="1.0"?>
<Response>
    <CmdType>Catalog</CmdType>
    <SN>1</SN>
    <DeviceID>34020000001320000001</DeviceID>
    <Result>OK</Result>
</Response>`

	return fmt.Sprintf("SIP/2.0 200 OK\r\n"+
		"Via: SIP/2.0/TCP %s:5060;branch=z9hG4bK%s\r\n"+
		"From: <sip:%s@%s>;tag=%s\r\n"+
		"To: <sip:%s@%s>;tag=%s\r\n"+
		"Call-ID: %s\r\n"+
		"CSeq: 1 MESSAGE\r\n"+
		"Content-Type: Application/MANSCDP+xml\r\n"+
		"Max-Forwards: 70\r\n"+
		"User-Agent: IPC\r\n"+
		"Content-Length: %d\r\n\r\n%s",
		serverIP, generateCallID(),
		deviceID, serverIP, generateCallID(),
		deviceID, serverIP, generateCallID(),
		callID,
		len(body), body)
}

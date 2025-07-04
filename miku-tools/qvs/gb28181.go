package qvs

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"middle-source-analysis/config"
	"middle-source-analysis/util"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"

	"github.com/icholy/digest"
)

var (
	Conf *config.Config
)

func Gb() {

	// 服务器地址
	serverAddr := fmt.Sprintf("%s:55068", Conf.Ip)

	// 连接到服务器
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {

		return
	}
	defer conn.Close()

	// 要发送的数据包
	data := make([]byte, 1000000000) //[]byte("hello world, 111111111111111111111111111111111111111111111111111111")

	// 发送数据包
	for i := 0; i < 10000000; i++ {
		_, err = conn.Write(data)
		if err != nil {

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

				return
			}

			fmt.Printf("服务器返回: %s\n", string(buf[:n]))
		}
	*/
}

func GbCli() {
	// SIP服务器地址
	sipServer := fmt.Sprintf("%s:5061", Conf.Ip)

	// 设备信息
	deviceID := "34020000001320000001"
	realm := "3402000000"
	password := "123456"

	// 连接到SIP服务器
	conn, err := net.Dial("tcp", sipServer)
	if err != nil {

		return
	}
	defer conn.Close()
	fmt.Println("连接到SIP服务器成功")

	// 发送REGISTER请求
	callID := generateCallID()
	nonce := ""

	// 第一次注册请求（不带认证信息）
	registerReq := createRegisterRequest(deviceID, Conf.Ip, callID, "", "", "")
	_, err = conn.Write([]byte(registerReq))
	if err != nil {

		return
	}

	// 读取401响应获取nonce
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {

		return
	}
	response := string(buf[:n])
	nonce = extractNonce(response)

	// 计算认证信息
	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", deviceID, realm, password))
	ha2 := md5Hash(fmt.Sprintf("REGISTER:sip:%s@%s", deviceID, Conf.Ip))
	response = md5Hash(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	// 发送带认证信息的REGISTER请求
	authenticatedRegister := createRegisterRequest(deviceID, Conf.Ip, callID, realm, nonce, response)
	_, err = conn.Write([]byte(authenticatedRegister))
	if err != nil {

		return
	}
	fmt.Println("注册成功")

	// 启动保活协程
	go func() {
		for {
			time.Sleep(60 * time.Second)
			keepaliveReq := createKeepAliveRequest(deviceID, Conf.Ip)
			_, err := conn.Write([]byte(keepaliveReq))
			if err != nil {

				return
			}
		}
	}()

	// 主循环处理接收到的消息
	for {
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {

			return
		}

		message := string(buf[:n])
		if strings.Contains(message, "MESSAGE") && strings.Contains(message, "Catalog") {
			fmt.Println("收到Catalog请求")
			// 处理Catalog请求
			catalogResponse := createCatalogResponse(deviceID, Conf.Ip, extractCallID(message))
			_, err = conn.Write([]byte(catalogResponse))
			if err != nil {

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

type SipSession struct {
	ID           string   `json:"id"`
	DeviceSumnum int      `json:"device_sumnum"`
	Devices      []Device `json:"devices"`
}

type Device struct {
	DeviceID     string `json:"device_id"`
	DeviceStatus string `json:"device_status"`
	InviteStatus string `json:"invite_status"`
	InviteTime   int64  `json:"invite_time"`
}

var totalChCnt int

func SipSess() {
	// 构建请求URL
	url := fmt.Sprintf("http://%s:%d/api/v1/gb28181?action=sip_query_session", Conf.Ip, Conf.Port)

	// 发送GET请求
	resp, err := http.Get(url)
	if err != nil {

		return
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {

		return
	}
	// 解析响应JSON
	result := struct {
		Data struct {
			Sessions []SipSession `json:"sessions"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(body, &result); err != nil {

		return
	}

	fmt.Printf("设备总数: %d, %s:%d\n", len(result.Data.Sessions), Conf.Ip, Conf.Port)
	for _, session := range result.Data.Sessions {
		totalChCnt += len(session.Devices)
		if len(session.Devices) < 50 {
			continue
		}
		fmt.Printf("Session ID: %s, 通道总数: %d, %s:%d\n", session.ID, session.DeviceSumnum, Conf.Ip, Conf.Port)
	}
}

func AllSipService() {
	ips := []string{"10.70.60.32", "10.70.67.38", "10.70.60.22"} // 77/75/76
	ports := []int{7279, 7272, 7273}                             // sip1/sip2/sip3
	for _, ip := range ips {
		for _, port := range ports {
			Conf.Ip = ip
			Conf.Port = port
			SipSess()
		}
	}
	fmt.Println("totalChCnt:", totalChCnt)
}

func Talk() {
	deleteAudioChannel()
	ssrc := createAudioChannel("127.0.0.1")
	if ssrc == -1 {
		return
	}
	code := sipInvite(int(ssrc), true, "127.0.0.1", 9015)
	if code != 0 {
		return
	}
	time.Sleep(1 * time.Second)
	// 每隔3秒发送一次音频PCM数据
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sendAppendAudioPCMRequest(); err != nil {

			}
		}
	}
}

type SRSMediaCreateChannelResponse struct {
	Code int `json:"code"`
	Data struct {
		Query struct {
			ID       string `json:"id"`
			IP       string `json:"ip"`
			RtmpPort int    `json:"rtmp_port"`
			App      string `json:"app"`
			Stream   string `json:"stream"`
			RtpPort  int    `json:"rtp_port"`
			TcpPort  int    `json:"tcp_port"`
			Ssrc     uint32 `json:"ssrc"`
			NodeId   string `json:"nodeId"`
		} `json:"query"`
	} `json:"data"`
}

func createAudioChannel(peer_ip string) int {
	// 构建请求URL
	url := fmt.Sprintf("http://127.0.0.1:2985/api/v1/gb28181?action=create_audio_channel&id=%s&app=live&protocol=tcp&enable_jitter_buf=true&peer_ip=%s", Conf.ID, peer_ip)

	// 发送GET请求
	resp, err := http.Get(url)
	if err != nil {
		log.Println("请求失败:", err)
		return -1
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("读取响应失败:", err)
		return -1
	}

	// 打印响应内容
	log.Println("createAudioChannel, resp:", string(body))
	r := SRSMediaCreateChannelResponse{}
	if err := json.Unmarshal(body, &r); err != nil {
		log.Println("解析JSON失败:", err)
		return -1
	}
	if r.Code != 0 && r.Code != 6001 {
		return -1
	}
	log.Println("ssrc:", r.Data.Query.Ssrc)
	return int(r.Data.Query.Ssrc)
}

type InviteResp struct {
	Code   int    `json:"code"`
	CallID string `json:"call_id"`
}

// 以下代码实现了使用 Go 语言发送 HTTP GET 请求到指定地址
func sipInvite(ssrc int, is_talk bool, ip string, port int) int {
	// 构建请求 URL
	url := fmt.Sprintf("http://localhost:7279/api/v1/gb28181?action=sip_invite&chid=%s&id=%s&ip=%s&rtp_port=%d&rtp_proto=tcp&is_talk=%v",
		Conf.ID, Conf.ID, ip, port, is_talk)
	url += fmt.Sprintf("&ssrc=%d", ssrc)

	// 发送 GET 请求
	resp, err := http.Get(url)
	if err != nil {
		log.Println("sipInvite, 请求失败:", err)
		return -1
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("sipInvite, 读取响应失败:", err)
		return -1
	}

	// 打印响应内容
	log.Println("sipInvite, resp:", string(body))
	r := InviteResp{}
	if err := json.Unmarshal(body, &r); err != nil {
		log.Println("sipInvite, 解析JSON失败:", err)
		return -1
	}

	return r.Code
}

// 定义一个函数用于发送指定的HTTP请求
func sendAppendAudioPCMRequest() error {
	// 定义请求的URL
	url := fmt.Sprintf("http://127.0.0.1:2985/api/v1/gb28181?action=append_audio_pcm&id=%s", Conf.ID)
	// 定义请求的数据
	data := []byte(`{
        "base64_pcm":"MTIzNDU2Cg=="
    }`)
	// 创建一个新的HTTP POST请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	// 创建一个HTTP客户端
	client := &http.Client{}
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	// 确保响应体在函数结束时关闭
	defer resp.Body.Close()
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 打印响应内容
	fmt.Println(string(body))
	return nil
}

func deleteAudioChannel() {
	url := fmt.Sprintf("http://127.0.0.1:2985/api/v1/gb28181?action=delete_audio_channel&id=%s", Conf.ID)
	// 发送GET请求
	resp, err := http.Get(url) // 发送GET请求
	if err != nil {
		log.Println("请求失败:", err)
		return
	}
	defer resp.Body.Close()
	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("读取响应失败:", err)
		return
	}
	// 打印响应内容
	log.Println("deleteAudioChannel, resp:", string(body))
}

func deleteVedioChannel(ip string, port int) {
	url := fmt.Sprintf("http://%s:%d/api/v1/gb28181?action=delete_channel&id=%s", ip, port, Conf.ID)
	// 发送GET请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Println("构建请求失败:", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("请求失败:", err)
		return
	}
	defer resp.Body.Close()
	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("读取响应失败:", err)
		return
	}
	// 打印响应内容
	log.Println("deleteVedioChannel, resp:", string(body))
}

func createVideoChannel(ip string, port int) int {
	// 构建请求URL
	url := fmt.Sprintf("http://%s:%d/api/v1/gb28181?action=create_channel&id=%s&app=live&protocol=tcp&enable_jitter_buf=true", ip, port, Conf.ID)

	// 发送GET请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Println("构建请求失败:", err)
		return -1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("请求失败:", err)
		return -1
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("读取响应失败:", err)
		return -1
	}

	// 打印响应内容
	log.Println("creatVideoChannel, resp:", string(body))
	r := SRSMediaCreateChannelResponse{}
	if err := json.Unmarshal(body, &r); err != nil {
		log.Println("解析JSON失败:", err)
		return -1
	}
	if r.Code != 0 && r.Code != 6001 {
		return -1
	}
	log.Println("ssrc:", r.Data.Query.Ssrc)
	return int(r.Data.Query.Ssrc)
}

func Invite() {
	deleteVedioChannel("101.133.131.188", 2985)
	ssrc := createVideoChannel("101.133.131.188", 2985)
	if ssrc == -1 {
		return
	}
	code := sipInvite(ssrc, false, "101.133.131.188", 9001)
	if code != 0 {
		return
	}
}

func sendTalkRtp(ssrc int) {
	conn, err := net.Dial("tcp", "127.0.0.1:9015")
	if err != nil {
		log.Println("连接失败:", err)
		return
	}
	//defer conn.Close()

	rtpPacket := util.RtpEnc([]byte("1234567890"), 1, 0, true, ssrc, 0)
	_, err = conn.Write(rtpPacket)
	if err != nil {
		log.Println("发送rtp包失败:", err)
		return
	}
	log.Println("send talk rtp success")

}

func sendVideoRtp(ssrc int) {
	conn, err := net.Dial("tcp", "127.0.0.1:9001")
	if err != nil {
		log.Println("连接失败:", err)
		return
	}
	//defer conn.Close()

	rtpPacket := util.RtpEnc([]byte("1234567890"), 1, 0, true, ssrc, 0)
	_, err = conn.Write(rtpPacket)
	if err != nil {
		log.Println("发送rtp包失败:", err)
		return
	}
	log.Println("send video rtp success")

}

func TalkTest() {
	log.Println("talk test")
	deleteAudioChannel()
	ssrc := createAudioChannel("127.0.0.2")
	if ssrc == -1 {
		return
	}
	sipInvite(ssrc, true, "127.0.0.1", 9015)
	time.Sleep(1 * time.Second)
	sendTalkRtp(ssrc)
	if err := sendAppendAudioPCMRequest(); err != nil {

	}
}

func uac(transport string) {
	inter1 := "100.100.8.254"
	inter := &inter1
	dst1 := fmt.Sprintf("%s:%d", Conf.Ip, Conf.Port)
	dst := &dst1
	tran := &transport
	username := &Conf.User
	password := &Conf.Passwd
	user := username

	// Make SIP Debugging available
	sip.SIPDebug = os.Getenv("SIP_DEBUG") != ""
	log := getLogger()

	// Setup UAC
	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent(*user),
	)
	if err != nil {
		log.Error("Fail to setup user agent", "error", err)
		return
	}

	client, err := sipgo.NewClient(ua, sipgo.WithClientHostname(*inter))
	if err != nil {
		log.Error("Fail to setup client handle", "error", err)
		return
	}
	defer client.Close()

	// Create basic REGISTER request structure
	recipient := sip.Uri{}
	sip.ParseUri(fmt.Sprintf("sip:%s@%s", *user, *dst), &recipient)
	req := sip.NewRequest(sip.REGISTER, recipient)
	req.AppendHeader(
		sip.NewHeader("Contact", fmt.Sprintf("<sip:%s@%s>", *username, *inter)),
	)
	req.SetTransport(strings.ToUpper(*tran))

	// Send request and parse response
	// req.SetDestination(*dst)
	log.Info(req.StartLine())
	ctx := context.Background()
	tx, err := client.TransactionRequest(ctx, req, sipgo.ClientRequestRegisterBuild)
	if err != nil {
		log.Error("Fail to create transaction", "error", err)
		return
	}
	defer tx.Terminate()

	res, err := getResponse(tx)
	if err != nil {
		log.Error("Fail to get response", "error", err)
		return
	}

	log.Info("Received status", "status", int(res.StatusCode))
	if res.StatusCode == 401 {
		// Get WwW-Authenticate
		wwwAuth := res.GetHeader("WWW-Authenticate")
		chal, err := digest.ParseChallenge(wwwAuth.Value())
		if err != nil {
			log.Error("Fail to parse challenge", "error", err, "wwwauth", wwwAuth.Value)
			return
		}

		// Reply with digest
		cred, _ := digest.Digest(chal, digest.Options{
			Method:   req.Method.String(),
			URI:      recipient.Host,
			Username: *username,
			Password: *password,
		})

		newReq := req.Clone()
		newReq.RemoveHeader("Via") // Must be regenerated by tranport layer
		newReq.AppendHeader(sip.NewHeader("Authorization", cred.String()))

		ctx := context.Background()
		tx, err := client.TransactionRequest(ctx, newReq, sipgo.ClientRequestAddVia)
		if err != nil {
			log.Error("Fail to create transaction", "error", err)
			return
		}
		defer tx.Terminate()

		res, err = getResponse(tx)
		if err != nil {
			log.Error("Fail to get response", "error", err)
			return
		}
	}

	if res.StatusCode != 200 {
		log.Error("Fail to register")
		return
	}

	log.Info("Client registered")
}

func getResponse(tx sip.ClientTransaction) (*sip.Response, error) {
	select {
	case <-tx.Done():
		return nil, fmt.Errorf("transaction died")
	case res := <-tx.Responses():
		return res, nil
	}
}

func getLogger() *slog.Logger {
	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	// zlog := zerolog.New(zerolog.ConsoleWriter{
	// 	Out:        oStdout,
	// 	TimeFormat: time.StampMicro,
	// }).With().Timestamp().Logger().Level(zerolog.InfoLevel)

	// logger := slog.New(slogzerolog.Option{Level: lvl, Logger: &zlog}.NewZerologHandler())
	// h := slog.NewTextHandler(oStdin, &slog.HandlerOptions{Level: lvl})
	// slog.SetDefault(slog.New(h))
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))); err != nil {
		lvl = slog.LevelInfo
	}
	slog.SetLogLoggerLevel(lvl)
	return slog.Default()
}

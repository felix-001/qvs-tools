package staging

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	mrand "math/rand"
	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type DnsResponse struct {
	Code      string `json:"code"`
	ConnectID string `json:"connectId"`
	Dns       []struct {
		Domain string   `json:"domain"`
		IPs    []string `json:"ips"`
		IPsV6  []string `json:"ipsv6"`
		TTL    int      `json:"ttl"`
	} `json:"dns"`
	Message string `json:"message"`
	UserIP  string `json:"userIp"`
	UserISP string `json:"userIsp"`
	UserLoc string `json:"userLoc"`
}

func TestDns(conf *config.Config, res *resources.Resources) {
	if conf.Uid != "" {
		conf.Ak, conf.Sk = util.GetAkSk(conf)
	} else {
		log.Println("need uid")
		return
	}
	if conf.Domain == "" {
		log.Println("need domain")
		return
	}
	for i := 0; i < conf.Loop; i++ {
		for isp, provIp := range res.V4Ips {
			for prov, ip := range provIp {
				addr := fmt.Sprintf("http://%s/?dns&domain=www.qiniu.com&ip=%s&type=0", conf.Domain, ip)
				respStr, err := util.QnHttpReq("GET", addr, "", conf.Ak, conf.Sk, map[string]string{})
				if err != nil {
				}
				resp := &DnsResponse{}
				if err := json.Unmarshal([]byte(respStr), resp); err != nil {
					log.Println("unmarshal err", err, resp, respStr)
					continue
				}
				if resp.Code != "1000" {
					log.Printf("dns test failed isp:%s prov:%s ip:%s code:%s message:%s\n",
						isp, prov, ip, resp.Code, resp.Message)
					continue
				}
				if len(resp.Dns) == 0 {
					log.Printf("dns test no dns records isp:%s prov:%s ip:%s\n", isp, prov, ip)
					continue
				}
				if len(resp.Dns[0].IPs) == 0 {
					log.Printf("dns test no ipv4 records isp:%s prov:%s ip:%s\n", isp, prov, ip)
					continue
				}
				for _, ipv4 := range resp.Dns[0].IPs {
					if ipv4 == conf.Ip {
						log.Println("found ip isp:", isp, "prov:", prov, "ip:", ip, "ipv4:", ipv4)
					}
				}
			}
		}
	}

}

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func RandAlphaNum(n int) (string, error) {
	b := make([]byte, n)
	l := big.NewInt(int64(len(letters)))
	for i := range b {
		r, err := rand.Int(rand.Reader, l)
		if err != nil {
			return "", err
		}
		b[i] = letters[r.Int64()]
	}
	return string(b), nil
}

func TestIqiyi(conf *config.Config, res *resources.Resources) {
	for range 100 {
		streamId, err := RandAlphaNum(15)
		if err != nil {
			log.Println("rand alphanum err", err)
			return
		}
		area := get302(streamId, res)
		log.Println("iqiyi test done, streamId:", streamId, " area:", area)
	}
}

func get302(streamId string, res *resources.Resources) string {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 返回此错误会停止重定向，返回原始响应
			return http.ErrUseLastResponse
		},
	}
	// 发起 HTTP 请求
	resp, err := client.Get(fmt.Sprintf("http://flv-qnplay.inter.71edge.com/live/%s.flv", streamId))
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// 检查是否为 302 重定向
	if resp.StatusCode != http.StatusFound {
		fmt.Printf("未收到 302 响应，状态码: %d, resp: %+v\n", resp.StatusCode, resp)
		return ""
	}

	// 解析 Location 头部
	location := resp.Header.Get("Location")
	if location == "" {
		fmt.Println("未找到 Location 头部")
		return ""
	}

	// 提取 IP 地址
	parsedURL, err := url.Parse(location)
	if err != nil {
		fmt.Printf("解析 URL 失败: %v\n", err)
		return ""
	}

	host := parsedURL.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	fmt.Printf("提取的 IP 地址: %s, location: %s\n", host, location)
	_, area, _ := util.GetLocate(host, res.IpParser)
	//log.Println(isp, area, region)
	return area
}

func sendRegister(gbid string, conn net.Conn) error {
	// SIP REGISTER message
	message := "REGISTER sip:101.133.131.188 SIP/2.0\r\n" +
		"Via: SIP/2.0/UDP 192.168.1.100:5060;branch=z9hG4bK776asdhds\r\n" +
		"Max-Forwards: 70\r\n" +
		"To: <sip:31011500002000000001@101.133.131.188>\r\n" +
		"From: <sip:" + gbid + "@101.133.131.188>;tag=1928301774\r\n" +
		"Call-ID: a84b4c76e66710\r\n" +
		"CSeq: 1 REGISTER\r\n" +
		"Contact: <sip:51010000991320705711@192.168.1.100:5060>\r\n" +
		"Expires: 3600\r\n" +
		"Content-Length: 0\r\n\r\n"
		// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	// Send SIP message
	_, err := conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending:", err)
		return err
	}
	return nil
}

func recv401(conf *config.Config, conn net.Conn) (string, error) {
	buf := make([]byte, 10240)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err)
		return "", err
	}
	if conf.Detail {
		log.Println("sip resp:", string(buf))
	}
	if !strings.Contains(string(buf), "401 Unauthorized") {
		log.Println("sip resp not 401")
		return "", fmt.Errorf("sip resp not 401")
	}
	return string(buf), nil
}

func sendRegisterWithAuth(sip401, gbid string, conn net.Conn) error {
	// Extract nonce from WWW-Authenticate header
	authHeader := ""
	for _, line := range strings.Split(string(sip401), "\r\n") {
		if strings.HasPrefix(line, "WWW-Authenticate:") {
			authHeader = line
			break
		}
	}
	if authHeader == "" {
		log.Println("No WWW-Authenticate header found")
		return fmt.Errorf("No WWW-Authenticate header found")
	}

	// 发送带鉴权信息的register
	// Parse nonce (simplified example - adjust according to your auth scheme)
	nonce := ""
	if strings.Contains(authHeader, "nonce=") {
		parts := strings.Split(authHeader, "nonce=")
		if len(parts) > 1 {
			nonce = strings.Split(parts[1], "\"")[1]
		}
	}

	// Create authenticated REGISTER message
	authMessage := "REGISTER sip:131011500002000000001 SIP/2.0\r\n" +
		"Via: SIP/2.0/UDP 192.168.1.100:5060;branch=z9hG4bK776asdhds\r\n" +
		"Max-Forwards: 70\r\n" +
		"To: <sip:31011500002000000001@101.133.131.188>\r\n" +
		"From: <sip:" + gbid + "@101.133.131.188>;tag=1928301774\r\n" +
		"Call-ID: a84b4c76e66710\r\n" +
		"CSeq: 2 REGISTER\r\n" +
		"Contact: <sip:51010000991320705711@192.168.1.100:5060>\r\n" +
		"Expires: 3600\r\n" +
		"Authorization: Digest username=\"user\", realm=\"\", nonce=\"" + nonce + "\", uri=\"sip:101.133.131.188\", response=\"\"\r\n" +
		"Content-Length: 0\r\n\r\n"

	// Send authenticated REGISTER
	_, err := conn.Write([]byte(authMessage))
	if err != nil {
		fmt.Println("Error sending authenticated REGISTER:", err)
		return err
	}
	return nil
}

func read200(conf *config.Config, conn net.Conn) error {
	// Read response to authenticated REGISTER
	buf := make([]byte, 10240)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return err
	}
	if conf.Detail {
		log.Println("Auth response:", string(buf))
	}
	if !strings.Contains(string(buf), "200 OK") {
		log.Println("sip resp not 200")
		return fmt.Errorf("sip resp not 200")
	}
	return nil
}

func runGbCli(conf *config.Config, gbid string) {
	conn, err := net.Dial("tcp", "101.133.131.188:55063")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()

	// 发送register
	if err := sendRegister(gbid, conn); err != nil {
		log.Println("Error sending:", err)
		return
	}
	log.Println("SIP REGISTER message sent successfully, gbid:", gbid)

	// 设备回复401
	sip401, err := recv401(conf, conn)
	if err != nil {
		fmt.Println("Error receiving:", err)
		return
	}
	log.Println("sip resp 401")

	// 发送带鉴权信息的register
	log.Println("Authenticated REGISTER sent, gbid:", gbid)
	if err := sendRegisterWithAuth(sip401, gbid, conn); err != nil {
		log.Println("Error sending:", err)
		return
	}

	// 设备回复200
	if err := read200(conf, conn); err != nil {
		log.Println("Error receiving:", err)
		return
	}
	log.Println("sip resp 200, gbid:", gbid)
	time.Sleep(time.Second * 600)
	for {
		if err := sendSipKeepalive(gbid, conn); err != nil {
			log.Println("Error sending:", err)
			//return
		}
		log.Println("SIP KEEPALIVE message sent successfully, gbid:", gbid)
		if err := read200(conf, conn); err != nil {
			log.Println("Error receiving:", err)
			//return
		}
		log.Println("sip keepalive resp 200, gbid:", gbid)
		if err := recvCatalog(conf, conn); err != nil {
			log.Println("Error receiving:", err)
			//return
		}
		log.Println("got sip catalog req, gbid:", gbid)
		if err := sendCatalogResp(gbid, conn); err != nil {
			log.Println("Error sending:", err)
			//return
		}
		log.Println("sip catalog resp sent, gbid:", gbid)
	}
}

func sendCatalogResp(gbid string, conn net.Conn) error {
	resp := fmt.Sprintf("MESSAGE sip:%s@%s SIP/2.0\r\n"+
		"Via: SIP/2.0/UDP %s\r\n"+
		"From: <sip:%s@%s>;tag=%s\r\n"+
		"To: <sip:%s@%s>\r\n"+
		"Call-ID: %s\r\n"+
		"CSeq: 1 MESSAGE\r\n"+
		"Content-Type: Application/MANSCDP+xml\r\n"+
		"Content-Length: 0\r\n\r\n",
		gbid, "domain.com",
		"local.ip:5060",
		"username", "domain.com", "123456",
		gbid, "domain.com",
		"call-id-123")

	_, err := conn.Write([]byte(resp))
	if err != nil {
		return fmt.Errorf("failed to send catalog response: %v", err)
	}
	return nil
}

func recvCatalog(conf *config.Config, conn net.Conn) error {
	buf := make([]byte, 10240)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err)
		return err

	}
	if conf.Detail {
		log.Println("sip req:", string(buf))
	}
	if !strings.Contains(string(buf), "MESSAGE") {
		log.Println("sip req not MESSAGE")
		return fmt.Errorf("sip req not MESSAGE")
	}
	if !strings.Contains(string(buf), "Catalog") {
		return fmt.Errorf("sip req not Catalog")
	}
	return nil
}

func sendSipKeepalive(gbid string, conn net.Conn) error {
	keepaliveMsg := fmt.Sprintf("MESSAGE sip:%s@%s SIP/2.0\r\n"+
		"Via: SIP/2.0/UDP %s;branch=z9hG4bK%s\r\n"+
		"From: <sip:%s@%s>;tag=%s\r\n"+
		"To: <sip:%s@%s>\r\n"+
		"Call-ID: %s\r\n"+
		"CSeq: 1 MESSAGE\r\n"+
		"Contact: <sip:%s@%s>\r\n"+
		"Content-Length: 0\r\n\r\n",
		gbid, conn.LocalAddr().String(),
		conn.LocalAddr().String(), generateRandomString(),
		gbid, conn.LocalAddr().String(), generateRandomString(),
		gbid, conn.LocalAddr().String(),
		generateRandomString(),
		gbid, conn.LocalAddr().String())

	_, err := conn.Write([]byte(keepaliveMsg))
	if err != nil {
		return fmt.Errorf("failed to send KEEPALIVE")
	}
	return nil
}

func generateRandomString() string {
	// 创建本地随机数生成器避免并发问题
	localRand := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	const charset = "0123456789"
	b := make([]byte, 20)
	for i := range b {
		b[i] = charset[localRand.Intn(len(charset))]
	}
	return string(b)
}

func TestSip(conf *config.Config) {
	// 创建本地随机数生成器
	//localRand := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	for i := 0; i < conf.N; i++ {
		//go func() {
		gbid := generateRandomString()
		go runGbCli(conf, gbid)
		//randomNum := localRand.Intn(3) + 1 // Generates random number between 1-5
		//log.Println("randomNum:", randomNum, "gbid:", gbid)
		time.Sleep(time.Second * time.Duration(1))
		runGbCli(conf, gbid)
		//}()
	}
	time.Sleep(time.Second * 10000)
}

func TestHy1(conf *config.Config) {
	file, err := os.Open("/Users/rigensen/Downloads/sqllab_liyqhy_20251202T122609.csv")
	if err != nil {
		log.Println("打开CSV文件失败:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // 允许列数不一致

	// 四个map：cdn和client各两个
	cdnBadMap := make(map[string]bool)    // cdnip -> field_video_bad_quality=="100"
	cdnAllMap := make(map[string]bool)    // 所有cdnip
	clientBadMap := make(map[string]bool) // clientIp -> field_video_bad_quality=="100"
	clientAllMap := make(map[string]bool) // 所有clientIp

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		if len(record) < 16 {
			log.Println("记录字段数不足16:", record)
			continue
		}

		// 第6列cdnip
		cdnip := strings.TrimSpace(record[5])
		if cdnip == "" {
			// 第14列url
			rawURL := strings.TrimSpace(record[13])
			if rawURL != "" {
				if u, err := url.Parse(rawURL); err == nil {
					host := u.Host
					if h, _, err := net.SplitHostPort(host); err == nil {
						cdnip = h
					} else {
						cdnip = host
					}
				} else {
					log.Println("url解析失败:", rawURL)
				}
			}
		}

		// 第16列field_video_bad_quality
		field_video_bad_quality := strings.TrimSpace(record[15])

		// 第3列clientIp
		clientIp := strings.TrimSpace(record[2])

		// 处理cdn
		if cdnip != "" {
			cdnAllMap[cdnip] = true
			if field_video_bad_quality == "100" {
				cdnBadMap[cdnip] = true
			}
		}

		// 处理client
		if clientIp != "" {
			clientAllMap[clientIp] = true
			if field_video_bad_quality == "100" {
				clientBadMap[clientIp] = true
			}
		} else {
			log.Println("clientIp为空")
		}
	}

	// 计算百分比
	cdnBadCount := len(cdnBadMap)
	cdnTotal := len(cdnAllMap)
	cdnPercent := 0.0
	if cdnTotal > 0 {
		cdnPercent = float64(cdnBadCount) / float64(cdnTotal) * 100
	}

	clientBadCount := len(clientBadMap)
	clientTotal := len(clientAllMap)
	clientPercent := 0.0
	if clientTotal > 0 {
		clientPercent = float64(clientBadCount) / float64(clientTotal) * 100
	}

	fmt.Printf("CDN: bad/total = %d/%d, 占比: %.2f%%\n", cdnBadCount, cdnTotal, cdnPercent)
	fmt.Printf("Client: bad/total = %d/%d, 占比: %.2f%%\n", clientBadCount, clientTotal, clientPercent)
}

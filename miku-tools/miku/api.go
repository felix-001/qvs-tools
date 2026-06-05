package miku

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	publicUtil "github.com/qbox/mikud-live/common/util"
)

type PlaycheckReq struct {
	Bucket     string            `json:"bucket"`
	Key        string            `json:"key"`
	Url        string            `json:"url"`
	Remote     string            `json:"remoteAddr"`
	Local      string            `json:"localAddr"`
	Node       string            `json:"nodeId"`
	ConnId     string            `json:"connectId"`
	Master     string            `json:"master"`
	Protocol   string            `json:"protocol"` // 协议类型，非必填，目前仅hls协议传递
	Headers    map[string]string `json:"headers"`
	User       string            `json:"user"`
	HttpMethod string            `json:"httpMethod"`
}

type PlayCheckResp struct {
	Uid        uint32 `json:"uid"`
	ErrCode    string `json:"code"`
	Message    string `json:"message"`
	ConnectId  string `json:"connectId"`
	Url302     string `json:"redirectUrl"`
	RewriteUrl string `json:"rewriteUrl"`
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	RegTsM3u8  string `json:"regTsM3u8"`  // m3u8文件中ts格式，替换其中的${app}/${stream}/${file}
	FlowMethod int    `json:"flowMethod"` // 计量方式: 1: miku计量系统; 2: pili计量系统; 其它值miku&pili计量系统
}

func playcheck(ip string, conf *config.Config) *PlayCheckResp {
	scheme := "http"
	if conf.Https {
		scheme += "s"
	}
	node := conf.Node
	if node == "" {
		node = "cf1b9ef8-33e6-3569-80f8-85ba87e4039f-vdn-jsyz1-dls-1-87"
	}
	if conf.App == "" {
		conf.App = "live"
	}

	playUrl := fmt.Sprintf("%s://%s/%s/%s.%s?did=a75e6982-7538-4629-ad3c-fd0d60b1ba54&expire=0",
		scheme, conf.Domain, conf.App, conf.Stream, conf.Format)
	if conf.QnTestUrl != "" {
		playUrl += "&qnTestUrl=" + conf.QnTestUrl
	}
	if conf.Player != "" {
		playUrl += "&player=" + conf.Player
	}
	if conf.Redirect {
		playUrl = fmt.Sprintf("%s://127.0.0.1/%s/%s/%s.%s?did=a75e6982-7538-4629-ad3c-fd0d60b1ba54&expire=0",
			scheme, conf.Domain, conf.App, conf.Stream, conf.Format)
	}
	if conf.Internal {
		playUrl = fmt.Sprintf("%s://127.0.0.1/%s/%s.%s?did=a75e6982-7538-4629-ad3c-fd0d60b1ba54&expire=0&domain=%s",
			scheme, conf.App, conf.Stream, conf.Format, conf.Domain)
	}

	b := make([]byte, 10)
	rand.Read(b)
	conf.ConnId = hex.EncodeToString(b)

	req := PlaycheckReq{
		Bucket:     conf.Bucket,
		Key:        conf.Stream,
		Url:        playUrl,
		Node:       node,
		Remote:     ip,
		ConnId:     conf.ConnId,
		User:       conf.User,
		Protocol:   conf.Protocol,
		Local:      "127.0.0.1:1234",
		HttpMethod: "GET",
	}
	if conf.HeadReq {
		req.HttpMethod = "HEAD"
	}
	fmt.Printf("req: %+v\n", req)
	bytes, err := json.Marshal(&req)
	if err != nil {
		log.Println(err)
		return nil
	}
	var resp PlayCheckResp
	port := conf.Port
	if port == 0 {
		port = 6060
	}
	addr := fmt.Sprintf("http://%s:%d/api/v1/playcheck", conf.SchedIp, port)
	if err := util.Post(addr, string(bytes), &resp); err != nil {
		log.Println(err)
		return nil
	}
	return &resp
}

func Playcheck(conf *config.Config) {
	remote := conf.Ip + ":8080"
	if publicUtil.IsIPv6(conf.Ip) {
		remote = fmt.Sprintf("[%s]:8080", conf.Ip)
	}
	resp := playcheck(remote, conf)
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

type Miku struct {
	conf      *config.Config
	resources resources.Resources
}

func NewMiku() *Miku {
	return &Miku{}
}

func (m *Miku) SetConf(conf *config.Config) {
	m.conf = conf
}

func (m *Miku) SetResources(resources resources.Resources) {
	m.resources = resources
}

func (m *Miku) DumpIps() {
	for isp, ips := range m.resources.V6Ips {
		fmt.Println(isp)
		for prov, ip := range ips {
			fmt.Printf("\tprov: %s, ip: %s\n", prov, ip)
		}
	}
	for isp, ips := range m.resources.V4Ips {
		fmt.Println(isp)
		for prov, ip := range ips {
			fmt.Printf("\tprov: %s, ip: %s\n", prov, ip)
		}
	}
}

func (m *Miku) LoopPlaycheck() {
	for _, ips := range m.resources.V6Ips {
		for _, ip := range ips {
			remote := ip + ":8080"
			if publicUtil.IsIPv6(ip) {
				remote = fmt.Sprintf("[%s]:8080", ip)
			}
			resp := playcheck(remote, m.conf)
			bytes, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				log.Println(err)
				return
			}
			fmt.Println(string(bytes))
			if publicUtil.IsIPv6(ip) {
				u, err := url.Parse(resp.Url302)
				if err != nil {
					log.Println(err)
					continue
				}
				// 处理 IPv6 地址，提取 IP 和端口
				if strings.HasPrefix(u.Host, "[") && strings.Contains(u.Host, "]") {
					idx := strings.Index(u.Host, "]")
					ip := u.Host[1:idx]
					if !publicUtil.IsIPv6(ip) {
						log.Println("invalid ip, not ipv6")
						continue
					}
				} else {
					log.Println("invalid ipv6", u.Host)
					continue
				}
			}
		}
	}

}

// TingYunErrNodes 发送 HTTP 请求获取听云网络数据
func (m *Miku) TingYunErrNodes() {
	if m.conf.Key == "" {
		log.Println("tingyun key is empty")
		return
	}
	// 若 StartTime 或 EndTime 为空，设置默认值
	if m.conf.StartTime == "" || m.conf.EndTime == "" {
		currentTime := time.Now()
		m.conf.EndTime = currentTime.Format("2006-01-02 15:04")
		m.conf.StartTime = currentTime.AddDate(0, 0, -1).Format("2006-01-02 15:04")
	}
	query := url.Values{}
	query.Add("authkey", m.conf.Key)
	query.Add("taskId", m.conf.Task)
	query.Add("beginTimeStr", m.conf.StartTime)
	query.Add("endTimeStr", m.conf.EndTime)
	query.Add("taskType", "3")
	addr := fmt.Sprintf("https://network.tingyun.com/network-report-data/rawdata/rawdata-csv-authkey?%s", query.Encode())
	log.Println("addr:", addr)
	resp, err := http.Get(addr)
	if err != nil {
		log.Printf("请求听云数据失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应数据失败: %v", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("请求失败，状态码: %d，响应内容: %s", resp.StatusCode, string(body))
		return
	}

	//fmt.Println(string(body))
	err = os.WriteFile("/tmp/tingyun.csv", body, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	m.statisticsErrNodes(string(body))
}

type ProbeResult struct {
	ClientIp string
	Locate   string
	WaitTime float64
	Time     string
}

type ClientipResult struct {
	PcdnIp   string
	Locate   string
	WaitTime float64
	Time     string
}

func (m *Miku) statisticsErrNodes(data string) {
	lines := strings.Split(data, "\n")
	log.Println("lines:", len(lines))
	ipResultsMap := make(map[string][]ProbeResult)
	clientIpResultMap := make(map[string][]ClientipResult)
	citypIspMap := make(map[string]int)
	pcdnIpAbnormalResultsMap := make(map[string][]ProbeResult)
	for _, line := range lines[1:] {
		fields := strings.Split(line, ",")
		if len(fields) < 27 {
			log.Println("fields:", len(fields))
			continue
		}
		// 去除 fields 中每个元素前后的双引号
		for i := range fields {
			fields[i] = strings.Trim(fields[i], "\"")
		}
		// 错误代码
		if fields[26] != "" {
			log.Println("err code:", fields[26])
			continue
		}
		waitTime, err := strconv.ParseFloat(fields[10], 64)
		if err != nil {
			log.Printf("将 fields[10] 转换为 float64 失败: %v", err)
			continue
		}
		pcdnIp := fields[7]
		ipResultsMap[pcdnIp] = append(ipResultsMap[pcdnIp], ProbeResult{
			ClientIp: fields[3],
			Locate:   fields[2],
			WaitTime: waitTime,
			Time:     fields[0],
		})
		clientIpResultMap[fields[3]] = append(clientIpResultMap[fields[3]], ClientipResult{
			PcdnIp:   pcdnIp,
			Locate:   fields[2],
			WaitTime: waitTime,
			Time:     fields[0],
		})
		citypIspMap[fields[2]]++
		if waitTime > 5 {
			pcdnIpAbnormalResultsMap[pcdnIp] = append(pcdnIpAbnormalResultsMap[pcdnIp], ProbeResult{
				ClientIp: fields[3],
				Locate:   fields[2],
				WaitTime: waitTime,
				Time:     fields[0],
			})
		}
	}
	log.Println("ipResultsMap:", len(ipResultsMap))
	for ip, results := range ipResultsMap {
		fmt.Printf("ip: %s, cnt: %d\n", ip, len(results))
	}
	ipAvgWaitTimeMap := make(map[string]float64)
	single := 0
	multi := 0
	for ip, results := range ipResultsMap {
		var totalWaitTime float64
		for _, result := range results {
			totalWaitTime += result.WaitTime
		}
		ipAvgWaitTimeMap[ip] = totalWaitTime / float64(len(results))
		if ipAvgWaitTimeMap[ip] > 10 {
			fmt.Printf("ip: %s, avg wait time: %.1f\n", ip, ipAvgWaitTimeMap[ip])
			for _, result := range results {
				fmt.Printf("\ttime: %s, client ip: %s, locate: %s, wait time: %.1f\n", result.Time, result.ClientIp, result.Locate, result.WaitTime)
			}
		}
		if len(results) >= 2 {
			multi++
		} else {
			single++
		}
	}
	log.Println("clientIpResultMap:", len(clientIpResultMap))
	for ip, results := range clientIpResultMap {
		fmt.Printf("client ip: %s, cnt: %d\n", ip, len(results))
		for _, result := range results {
			fmt.Printf("\ttime: %s, pcdn ip: %s, locate: %s, wait time: %.1f\n", result.Time, result.PcdnIp, result.Locate, result.WaitTime)
		}
	}
	log.Println("single:", single)
	log.Println("multi:", multi)
	log.Println("citypIspMap:", len(citypIspMap))
	for city, cnt := range citypIspMap {
		fmt.Printf("city: %s, cnt: %d\n", city, cnt)
	}
	log.Println("pcdnIpAbnormalResultsMap:", len(pcdnIpAbnormalResultsMap))
	for ip, results := range pcdnIpAbnormalResultsMap {
		if len(results) < 2 {
			continue
		}
		fmt.Printf("ip: %s, cnt: %d\n", ip, len(results))
		for _, result := range results {
			fmt.Printf("\ttime: %s, client ip: %s, locate: %s, wait time: %.1f\n", result.Time, result.ClientIp, result.Locate, result.WaitTime)
		}
	}
	tasks, err := m.getTaskList()
	if err != nil {
		log.Println("err:", err)
		return
	}
	for _, task := range tasks {
		fmt.Printf("task: %s, url: %s, id: %d, name: %s, expire: %s\n", task.Name, task.Url, task.ID, task.Name, task.Expire)
	}
}

func (m *Miku) IpLoc() {
	if m.conf.Ip == "" {
		log.Println("need -ip")
		return
	}
	country, isp, area, prov := util.GetLocate(m.conf.Ip, m.resources.IpParser)
	log.Println("country:", country)
	log.Println("isp:", isp)
	log.Println("area:", area)
	log.Println("prov:", prov)
	if m.conf.N > 0 {
		for i := 0; i < m.conf.N; i++ {
			country, isp, area, prov := util.GetLocate(m.conf.Ip, m.resources.IpParser)
			log.Println("country:", country)
			log.Println("isp:", isp)
			log.Println("area:", area)
			log.Println("prov:", prov)
			time.Sleep(time.Second)
		}
	}
}

type StreamRegisterRequest struct {
	Bucket string `json:"bucket" binding:"required"` // 空间
	Key    string `json:"key" binding:"required"`    // 流ID
	Node   string `json:"nodeId" binding:"required"` // nodeID
	Url    string `json:"url" binding:"required"`    // 完整推流url
	RawUrl string `json:"rawUrl"`                    // 改写之前的完整推流url
	Type   string `json:"type" binding:"required"`   // 业务类型(协议）:  live (rtmp | httpflv | hls ), rtc, iovt ( gb28181 | onvif )
	//Protocol  string `json:"protocol" binding:"required"` // 协议
	IP         string `json:"ip"` // 请求IP
	ConnectId  string `json:"connectId" binding:"required"`
	LocalAddr  string `json:"localAddr"`
	RemoteAddr string `json:"remoteAddr"`
	Domain     string `json:"domain" binding:"required"` // 推流域名
	EdgePort   string `json:"edgePort"`                  // 边缘节点对外开放的端口
	Master     string `json:"masterKey"`                 //publishcheck 如果streamConf开启主备流
}

func (m *Miku) StreamRegister() {
	conf := m.conf

	// 生成 ConnectId（如果为空）
	connId := conf.ConnId
	if connId == "" {
		b := make([]byte, 10)
		rand.Read(b)
		connId = hex.EncodeToString(b)
	}

	// 检查必填字段
	var missing []string
	if conf.Bucket == "" {
		missing = append(missing, "-bucket")
	}
	if conf.Stream == "" {
		missing = append(missing, "-stream")
	}
	if conf.Node == "" {
		missing = append(missing, "-node")
	}
	if conf.Url == "" {
		missing = append(missing, "-url")
	}
	if conf.Protocol == "" {
		missing = append(missing, "-protocol")
	}
	if conf.Domain == "" {
		missing = append(missing, "-domain")
	}
	if conf.Ip == "" {
		missing = append(missing, "-ip")
	}
	if len(missing) > 0 {
		log.Printf("缺少必填参数: %v", missing)
		return
	}

	req := StreamRegisterRequest{
		Bucket:    conf.Bucket,
		Key:       conf.Stream,
		Node:      conf.Node,
		Url:       conf.Url,
		RawUrl:    conf.RawApp,
		Type:      conf.Protocol,
		IP:        conf.Ip,
		ConnectId: connId,
		Domain:    conf.Domain,
		LocalAddr: fmt.Sprintf("%s:8080", conf.Ip),
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		log.Printf("序列化请求失败: %v", err)
		return
	}

	log.Printf("请求 StreamRegister: %s", string(reqBody))

	addr := fmt.Sprintf("http://%s:6060/api/v1/streamregister", m.conf.SchedIp)
	resp, err := http.Post(addr, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return
	}

	// 美化输出 JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBody, "", "  "); err != nil {
		// 如果不是合法 JSON，直接输出原始内容
		fmt.Printf("响应 (状态码 %d):\n%s\n", resp.StatusCode, string(respBody))
		return
	}
	fmt.Printf("响应 (状态码 %d):\n%s\n", resp.StatusCode, prettyJSON.String())
}

type HandshakeInfo struct {
	ClientHelloTime time.Time
	CCSTime         time.Time
	ClientIP        string
	ServerIP        string
}

func (m *Miku) Packet_backup() {
	handle, err := pcap.OpenOffline("/tmp/test.pcap")
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	handshakes := make(map[uint64]*HandshakeInfo) // key: tcp stream index
	packetNum := 0

	for packet := range packetSource.Packets() {
		packetNum++
		// 获取 TCP 层和网络层
		netLayer := packet.NetworkLayer()
		tcpLayer := packet.TransportLayer()
		if netLayer == nil || tcpLayer == nil {
			continue
		}

		// 获取 TCP 流的标识 (gopacket 内部哈希或者自己拼 5 元组)
		// 这里为了简化，使用 gopacket 自带的 Flow 标识
		flow := tcpLayer.TransportFlow().FastHash()
		// 注意：FastHash 是单向的，为了双向匹配，需要规范流Key
		// 这里略去复杂的流归一化逻辑，假设你知道如何处理

		payload := tcpLayer.LayerPayload()
		if len(payload) < 5 {
			continue // 不是 TLS 包
		}

		contentType := payload[0]
		// 1. 检测 Client Hello (Content Type: 22, Handshake Type: 1)
		if contentType == 22 && len(payload) > 5 && payload[5] == 1 {
			// 记录 Client Hello
			srcIP := netLayer.NetworkFlow().Src().String()
			dstIP := netLayer.NetworkFlow().Dst().String()
			handshakes[flow] = &HandshakeInfo{
				ClientHelloTime: packet.Metadata().Timestamp,
				ClientIP:        srcIP,
				ServerIP:        dstIP,
			}
		}

		// 2. 检测 Change Cipher Spec (Content Type: 20)
		if contentType == 20 {
			if info, exists := handshakes[flow]; exists {
				// 确保是服务端发回来的 CCS
				srcIP := netLayer.NetworkFlow().Src().String()
				if srcIP == info.ServerIP {
					info.CCSTime = packet.Metadata().Timestamp
					duration := info.CCSTime.Sub(info.ClientHelloTime).Milliseconds()
					fmt.Printf("Packet %d, Stream %d: Handshake took %d ms\n", packetNum, flow, duration)
					delete(handshakes, flow) // 计算完毕，移除
				}
			}
		}
	}
}

func (m *Miku) ParseCsv() {
	pcapFile := m.conf.F
	if pcapFile == "" {
		log.Println("缺少 pcap 文件路径，请使用 -f <pcap_file>")
		return
	}

	outputFile := "/tmp/tls_handshakes.csv"

	cmd := exec.Command("tshark",
		"-r", pcapFile,
		"-Y", "tls.handshake.type==1 or tls.record.content_type==20",
		"-T", "fields",
		"-e", "frame.number",
		"-e", "frame.time_relative",
		"-e", "tcp.stream",
		"-e", "ip.src",
		"-e", "ip.dst",
		"-e", "tls.handshake.type",
		"-e", "tls.record.content_type",
		"-E", "header=y",
		"-E", "separator=,",
	)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		log.Printf("执行 tshark 失败: %v, stderr: %s", err, stderr.String())
		return
	}

	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		log.Printf("写入输出文件失败: %v", err)
		return
	}

	log.Printf("tshark 分析完成，结果已保存到 %s", outputFile)
}

type TLSRecord struct {
	FrameNum    int
	Time        float64
	Stream      int
	Src         string
	Dst         string
	Handshake   float64
	ContentType float64
}

type HandshakeResult struct {
	Stream         int
	ClientHello    float64
	ServerCCS      float64
	Duration       float64
	ClientEndpoint string
	ServerEndpoint string
	FrameNum       int
}

func (m *Miku) Packet() {
	m.ParseCsv()
	// 读取 CSV
	file, err := os.Open("/tmp/tls_handshakes.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // tshark 可能省略尾部空字段，不校验字段数
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	if len(records) < 2 {
		log.Println("CSV 文件中没有数据")
		return
	}

	// 按流分组
	streams := make(map[int][]TLSRecord)

	// 跳过标题行
	for _, row := range records[1:] {
		// tshark 会省略尾部空字段，补齐到 7 个字段
		cols := make([]string, 7)
		for i, v := range row {
			if i < 7 {
				cols[i] = v
			}
		}

		frame, _ := strconv.Atoi(cols[0])
		t, _ := strconv.ParseFloat(cols[1], 64)
		stream, _ := strconv.Atoi(cols[2])

		handshake, _ := strconv.ParseFloat(cols[5], 64)
		contentType, _ := strconv.ParseFloat(cols[6], 64)

		record := TLSRecord{
			FrameNum:    frame,
			Time:        t,
			Stream:      stream,
			Src:         cols[3],
			Dst:         cols[4],
			Handshake:   handshake,
			ContentType: contentType,
		}

		streams[stream] = append(streams[stream], record)
	}

	var results []HandshakeResult

	// 分析每个流
	for stream, packets := range streams {
		// 按时间排序
		sort.Slice(packets, func(i, j int) bool {
			return packets[i].Time < packets[j].Time
		})

		// 找到 Client Hello
		var clientHelloTime float64
		var clientHelloSrc, clientHelloDst string

		for _, pkt := range packets {
			if pkt.Handshake == 1.0 {
				clientHelloTime = pkt.Time
				clientHelloSrc = pkt.Src
				clientHelloDst = pkt.Dst
				break
			}
		}

		if clientHelloTime == 0 {
			continue
		}

		// 找到服务器 Change Cipher Spec
		for _, pkt := range packets {
			if pkt.Time <= clientHelloTime {
				continue
			}

			// 检查是否来自服务器的 Change Cipher Spec
			if pkt.Handshake == 0 && pkt.ContentType == 20.0 &&
				pkt.Src == clientHelloDst && pkt.Dst == clientHelloSrc {

				duration := (pkt.Time - clientHelloTime) * 1000 // 转毫秒

				results = append(results, HandshakeResult{
					Stream:         stream,
					ClientHello:    clientHelloTime,
					ServerCCS:      pkt.Time,
					Duration:       duration,
					ClientEndpoint: fmt.Sprintf("%s -> %s", clientHelloSrc, clientHelloDst),
					ServerEndpoint: fmt.Sprintf("%s -> %s", pkt.Src, pkt.Dst),
					FrameNum:       pkt.FrameNum,
				})
				break
			}
		}
	}

	// 输出结果
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("SSL 握手耗时分析")
	fmt.Println(strings.Repeat("=", 80))

	var totalDuration float64
	for _, result := range results {
		fmt.Printf("TCP Stream %d:\n", result.Stream)
		fmt.Printf("  Client Hello 时间: %.6fs\n", result.ClientHello)
		fmt.Printf("  Change Cipher Spec 时间: %.6fs\n", result.ServerCCS)
		fmt.Printf("  握手耗时: %.2fms\n", result.Duration)
		fmt.Printf("  客户端 -> 服务器: %s\n", result.ClientEndpoint)
		fmt.Println()
		totalDuration += result.Duration
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("总握手次数: %d\n", len(results))
	if len(results) > 0 {
		fmt.Printf("平均握手耗时: %.2fms\n", totalDuration/float64(len(results)))
	}
}

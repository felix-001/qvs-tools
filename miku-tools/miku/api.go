package miku

import (
	"bytes"
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
	"strconv"
	"strings"
	"time"

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

	playUrl := fmt.Sprintf("%s://%s/%s/%s.%s",
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
	if conf.Query != "" {
		if strings.Contains(playUrl, "?") {
			playUrl += "&" + conf.Query
		} else {
			playUrl += "?" + conf.Query
		}
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

// TingYunErrNodes 请求听云API获取原始数据，分析再缓冲时间异常的IP
func (m *Miku) TingYunErrNodes() {
	// 获取 auth_key：命令行 -key 优先，否则从配置文件获取
	authKey := m.conf.Key
	if authKey == "" {
		authKey = m.conf.Tingyun.AuthKey
	}
	if authKey == "" {
		log.Println("tingyun auth key is empty, use -key or configure tingyun.auth_key")
		return
	}

	// 获取 task ID：命令行 -task 优先，否则从配置文件获取
	taskId := m.conf.Task
	if taskId == "" {
		taskId = m.conf.Tingyun.TaskId
	}
	if taskId == "" {
		log.Println("tingyun task id is empty, use -task or configure tingyun.task_id")
		return
	}

	// 获取域名：配置文件优先，默认 network.tingyun.com
	domain := m.conf.Tingyun.Domain
	if domain == "" {
		domain = "network.tingyun.com"
	}

	// 若 StartTime 或 EndTime 为空，设置默认值（最近24小时）
	if m.conf.StartTime == "" || m.conf.EndTime == "" {
		currentTime := time.Now()
		m.conf.EndTime = currentTime.Format("2006-01-02 15:04")
		m.conf.StartTime = currentTime.AddDate(0, 0, -1).Format("2006-01-02 15:04")
	}

	// 构建请求URL
	query := url.Values{}
	query.Add("authkey", authKey)
	query.Add("taskId", taskId)
	query.Add("beginTimeStr", m.conf.StartTime)
	query.Add("endTimeStr", m.conf.EndTime)
	query.Add("taskType", "3")
	addr := fmt.Sprintf("https://%s/network-report-data/rawdata/rawdata-csv-authkey?%s", domain, query.Encode())
	log.Println("addr:", addr)

	// 发送请求
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

	// 保存原始数据
	err = os.WriteFile("/tmp/tingyun.csv", body, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("原始数据已保存到 /tmp/tingyun.csv")

	m.analyzeRebufferTime(string(body))
}

// rebufferRecord 单条听云原始数据记录
type rebufferRecord struct {
	Time         string
	MonitorIP    string  // 监测点IP
	TargetIP     string  // 目标主机IP (含地区信息，如 "183.134.26.4(衢州>移动)")
	TargetIPPure string  // 目标主机IP (纯IP)
	City         string  // 城市
	Isp          string  // 运营商
	RebufferTime float64 // 再缓冲时间(s)
	CityIsp      string  // 城市运营商
}

// ipRebufferAnalysis 按IP分组的再缓冲时间分析结果
type ipRebufferAnalysis struct {
	IP              string
	TotalRecords    int
	AbnormalRecords int
	AbnormalRate    float64
	AbnormalDetails []rebufferRecord
}

func (m *Miku) analyzeRebufferTime(data string) {
	lines := strings.Split(data, "\n")
	if len(lines) < 2 {
		log.Println("no data")
		return
	}

	// 解析表头，获取列索引
	headerFields := strings.Split(lines[0], ",")
	for i := range headerFields {
		headerFields[i] = strings.Trim(headerFields[i], "\"")
	}
	colIndex := make(map[string]int)
	for i, h := range headerFields {
		colIndex[strings.TrimSpace(h)] = i
	}

	monitorIPCol, ok1 := colIndex["监测点 IP"]
	targetIPCol, ok2 := colIndex["目标主机 IP"]
	rebufferCol, ok3 := colIndex["再缓冲时间(s)"]
	if !ok1 || !ok2 || !ok3 {
		log.Printf("缺少必要列: 监测点IP=%v, 目标主机IP=%v, 再缓冲时间=%v", ok1, ok2, ok3)
		return
	}

	timeCol := colIndex["时间"]
	cityCol := colIndex["城市"]
	ispCol := colIndex["运营商"]
	cityIspCol := colIndex["城市运营商"]
	errCodeCol := colIndex["错误代码"]

	filterIP := m.conf.FilterIp

	// 按 监测点IP 和 目标主机IP 分组统计
	monitorIPMap := make(map[string]*ipRebufferAnalysis)
	targetIPMap := make(map[string]*ipRebufferAnalysis)

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		for i := range fields {
			fields[i] = strings.Trim(fields[i], "\"")
		}

		if len(fields) <= rebufferCol {
			continue
		}

		// 跳过有错误代码的记录
		if errCodeCol > 0 && errCodeCol < len(fields) && fields[errCodeCol] != "" {
			continue
		}

		monitorIP := fields[monitorIPCol]
		targetIP := fields[targetIPCol]
		rebufferTimeStr := fields[rebufferCol]

		rebufferTime, err := strconv.ParseFloat(rebufferTimeStr, 64)
		if err != nil {
			continue
		}

		// 如果指定了 filter_ip，只处理匹配的记录
		if filterIP != "" && monitorIP != filterIP && !strings.HasPrefix(targetIP, filterIP) {
			continue
		}

		// 提取纯目标主机IP（去掉括号中的地区信息）
		targetIPPure := targetIP
		if idx := strings.Index(targetIP, "("); idx > 0 {
			targetIPPure = targetIP[:idx]
		}

		city := ""
		if cityCol > 0 && cityCol < len(fields) {
			city = fields[cityCol]
		}
		isp := ""
		if ispCol > 0 && ispCol < len(fields) {
			isp = fields[ispCol]
		}
		timeStr := ""
		if timeCol >= 0 && timeCol < len(fields) {
			timeStr = fields[timeCol]
		}
		cityIsp := ""
		if cityIspCol > 0 && cityIspCol < len(fields) {
			cityIsp = fields[cityIspCol]
		}

		record := rebufferRecord{
			Time:         timeStr,
			MonitorIP:    monitorIP,
			TargetIP:     targetIP,
			TargetIPPure: targetIPPure,
			City:         city,
			Isp:          isp,
			RebufferTime: rebufferTime,
			CityIsp:      cityIsp,
		}

		isAbnormal := rebufferTime > 30

		// 按监测点IP分组
		if _, ok := monitorIPMap[monitorIP]; !ok {
			monitorIPMap[monitorIP] = &ipRebufferAnalysis{IP: monitorIP}
		}
		monitorIPMap[monitorIP].TotalRecords++
		if isAbnormal {
			monitorIPMap[monitorIP].AbnormalRecords++
			monitorIPMap[monitorIP].AbnormalDetails = append(monitorIPMap[monitorIP].AbnormalDetails, record)
		}

		// 按目标主机IP分组
		if _, ok := targetIPMap[targetIPPure]; !ok {
			targetIPMap[targetIPPure] = &ipRebufferAnalysis{IP: targetIPPure}
		}
		targetIPMap[targetIPPure].TotalRecords++
		if isAbnormal {
			targetIPMap[targetIPPure].AbnormalRecords++
			targetIPMap[targetIPPure].AbnormalDetails = append(targetIPMap[targetIPPure].AbnormalDetails, record)
		}
	}

	// 输出结果
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  听云再缓冲时间异常分析报告")
	fmt.Printf("  时间范围: %s ~ %s\n", m.conf.StartTime, m.conf.EndTime)
	if filterIP != "" {
		fmt.Printf("  过滤IP: %s\n", filterIP)
	}
	fmt.Println("  异常判定: 再缓冲时间 > 30s 为异常记录, 异常率 > 60% 为异常IP")
	fmt.Println("========================================")

	fmt.Println()
	fmt.Println("--- 监测点IP异常分析 ---")
	hasAbnormal := false
	for _, analysis := range monitorIPMap {
		if analysis.TotalRecords == 0 {
			continue
		}
		analysis.AbnormalRate = float64(analysis.AbnormalRecords) / float64(analysis.TotalRecords) * 100
		if analysis.AbnormalRate > 60 {
			hasAbnormal = true
			fmt.Printf("[异常] 监测点IP: %s, 总记录: %d, 异常记录: %d, 异常率: %.1f%%\n",
				analysis.IP, analysis.TotalRecords, analysis.AbnormalRecords, analysis.AbnormalRate)
			for _, r := range analysis.AbnormalDetails {
				fmt.Printf("  时间: %s, 目标主机: %s, 城市: %s, 运营商: %s, 再缓冲时间: %.3fs\n",
					r.Time, r.TargetIP, r.City, r.Isp, r.RebufferTime)
			}
		}
	}
	if !hasAbnormal {
		fmt.Println("未发现异常的监测点IP")
	}

	fmt.Println()
	fmt.Println("--- 目标主机IP异常分析 ---")
	hasAbnormal = false
	for _, analysis := range targetIPMap {
		if analysis.TotalRecords == 0 {
			continue
		}
		analysis.AbnormalRate = float64(analysis.AbnormalRecords) / float64(analysis.TotalRecords) * 100
		if analysis.AbnormalRate > 60 {
			hasAbnormal = true
			fmt.Printf("[异常] 目标主机IP: %s, 总记录: %d, 异常记录: %d, 异常率: %.1f%%\n",
				analysis.IP, analysis.TotalRecords, analysis.AbnormalRecords, analysis.AbnormalRate)
			for _, r := range analysis.AbnormalDetails {
				fmt.Printf("  时间: %s, 监测点IP: %s, 城市: %s, 运营商: %s, 再缓冲时间: %.3fs\n",
					r.Time, r.MonitorIP, r.City, r.Isp, r.RebufferTime)
			}
		}
	}
	if !hasAbnormal {
		fmt.Println("未发现异常的目标主机IP")
	}

	// 打印所有IP的摘要
	fmt.Println()
	fmt.Println("--- 全部监测点IP摘要 ---")
	for _, analysis := range monitorIPMap {
		if analysis.TotalRecords == 0 {
			continue
		}
		analysis.AbnormalRate = float64(analysis.AbnormalRecords) / float64(analysis.TotalRecords) * 100
		fmt.Printf("监测点IP: %s, 总记录: %d, 异常记录: %d, 异常率: %.1f%%\n",
			analysis.IP, analysis.TotalRecords, analysis.AbnormalRecords, analysis.AbnormalRate)
	}

	fmt.Println()
	fmt.Println("--- 全部目标主机IP摘要 ---")
	for _, analysis := range targetIPMap {
		if analysis.TotalRecords == 0 {
			continue
		}
		analysis.AbnormalRate = float64(analysis.AbnormalRecords) / float64(analysis.TotalRecords) * 100
		fmt.Printf("目标主机IP: %s, 总记录: %d, 异常记录: %d, 异常率: %.1f%%\n",
			analysis.IP, analysis.TotalRecords, analysis.AbnormalRecords, analysis.AbnormalRate)
	}

	// 打印可用任务列表
	tasks, err := m.getTaskList()
	if err != nil {
		log.Println("获取任务列表失败:", err)
		return
	}
	fmt.Println()
	fmt.Println("--- 可用任务列表 ---")
	for _, task := range tasks {
		fmt.Printf("任务: %s, ID: %d, URL: %s, 过期时间: %s\n", task.Name, task.ID, task.Url, task.Expire)
	}
}

func (m *Miku) IpLoc() {
	m.nodesIpLoc()
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

// ipLocNodeIds 需要循环查询地理位置信息的节点列表
var ipLocNodeIds = []string{
	"ea9f3e95-47f6-3b87-8e29-a3c60c31d3f8-yzh999",
	"9b836e81-a37d-3bcf-b41c-5b55d7718a0e-yzh996",
	"fa670e5e-5398-3e35-8380-198ec9f028a3-vdn-zjwz-dls-1-37",
	"b2d4f969-d7b8-3f87-8ac5-b017f9151ce6-vdn-zjwz-dls-1-38",
}

// nodesIpLoc 遍历 ipLocNodeIds，对每个节点获取其公网 IP 列表，
// 再查询 IP 库得到地理位置信息并打印。整个动作循环 -n 次。
func (m *Miku) nodesIpLoc() {
	apiBase := m.conf.Addr
	if apiBase == "" {
		apiBase = defaultOriginNodeAPI
	}

	loops := m.conf.N
	if loops <= 0 {
		loops = 1
	}

	for i := 0; i < loops; i++ {
		log.Printf("=== 节点地理位置查询 第 %d/%d 轮 ===", i+1, loops)
		for _, nodeId := range ipLocNodeIds {
			node, err := fetchRtNode(apiBase, nodeId)
			if err != nil {
				log.Printf("获取节点 %s 失败: %v", nodeId, err)
				continue
			}
			ips := getPublicIPs(node)
			if len(ips) == 0 {
				log.Printf("节点 %s 无公网 IP，跳过", nodeId)
				continue
			}
			log.Printf("节点 %s 公网 IP: %v", nodeId, ips)
			for _, ip := range ips {
				country, isp, area, prov := util.GetLocate(ip, m.resources.IpParser)
				log.Printf("  IP: %s, country: %s, isp: %s, area: %s, prov: %s",
					ip, country, isp, area, prov)
			}
		}
		if i < loops-1 {
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

	connId := conf.ConnId

	// 检查必填字段
	if conf.Node == "" {
		conf.Node = "cf1b9ef8-33e6-3569-80f8-85ba87e4039f-vdn-jsyz1-dls-1-87"
	}
	if conf.Url == "" {
		conf.Url = fmt.Sprintf("rtmp://%s/%s/teststream", conf.Bucket, conf.Domain)
	}
	if conf.ConnId == "" {
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, 10)
		for i := range b {
			b[i] = charset[rand.Intn(len(charset))]
		}
		connId = string(b)
		conf.ConnId = connId
	}

	req := StreamRegisterRequest{
		Bucket:     conf.Bucket,
		Key:        conf.Stream,
		Node:       conf.Node,
		Url:        conf.Url,
		RawUrl:     conf.RawApp,
		Type:       "live",
		IP:         conf.Ip,
		ConnectId:  connId,
		Domain:     conf.Domain,
		LocalAddr:  fmt.Sprintf("%s:1935", conf.Ip),
		RemoteAddr: "61.169.114.36:2345",
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

/*
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
*/

func (m *Miku) ParseCsv() {
	pcapFile := m.conf.F
	if pcapFile == "" {
		log.Println("缺少 pcap 文件路径，请使用 -f <pcap_file>")
		return
	}
	if m.conf.Ip == "" {
		log.Println("缺少 ip 参数，请使用 -ip <ip>")
		return
	}

	outputFile := "/tmp/tls_handshakes.csv"

	cmd := exec.Command("tshark",
		"-r", pcapFile,
		"-Y", fmt.Sprintf("ip.addr == %s and (tls.handshake.type==1 or tls.handshake.type==4)", m.conf.Ip),
		"-T", "fields",
		"-e", "frame.number",
		"-e", "frame.time_relative",
		"-e", "tcp.stream",
		"-e", "ip.src",
		"-e", "ip.dst",
		"-e", "tls.handshake.type",
		//"-e", "tls.record.content_type",
		"-E", "header=y",
		"-E", "separator=,",
		"-E", "aggregator=;", // 修改这里：用分号聚合同名字段
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

/*
func (m *Miku) Packet() {
	if _, err := os.Stat("/tmp/tls_handshakes.csv"); os.IsNotExist(err) {
		m.ParseCsv()
	}
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
		// tshark 会省略尾部空字段，补齐到 6 个字段
		cols := make([]string, 6)
		for i, v := range row {
			if i < 6 {
				cols[i] = v
			}
		}

		frame, _ := strconv.Atoi(cols[0])
		t, _ := strconv.ParseFloat(cols[1], 64)
		stream, _ := strconv.Atoi(cols[2])

		handshake, _ := strconv.ParseFloat(cols[5], 64)
		//contentType, _ := strconv.ParseFloat(cols[6], 64)

		record := TLSRecord{
			FrameNum:  frame,
			Time:      t,
			Stream:    stream,
			Src:       cols[3],
			Dst:       cols[4],
			Handshake: handshake,
			//ContentType: contentType,
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
			if pkt.Handshake == 4.0 {

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
		fmt.Printf("FrameNum: %d\n", result.FrameNum)
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
*/

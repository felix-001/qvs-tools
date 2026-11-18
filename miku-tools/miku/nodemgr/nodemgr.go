package nodemgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	commonModel "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

type NodeMgr struct {
	allRootNodesMapByNodeId map[string]*commonModel.RtNode
	allNodesMap             map[string]*commonModel.RtNode
	conf                    *config.Config
	resources               resources.Resources
	modules                 []NodeCallback
	filterMgr               *FilterMgr
}

func NewNodeMgr() *NodeMgr {
	conf := SwitchConf{
		Dynamic:        true,
		NotBanProv:     true,
		Serving:        true,
		NoStreamdPorts: true,
		Abilities:      true,
		Services:       true,
		TimeLimit:      true,
		PrivateIp:      true,
		Forbidden:      true,
		ProbeSpeed:     true,
		Ipv6:           false,
		Rtt:            true,
		Loss:           true,
		TcpRetrans:     true,
		AvailableBw:    false,
	}
	return &NodeMgr{
		allRootNodesMapByNodeId: make(map[string]*commonModel.RtNode),
		allNodesMap:             make(map[string]*commonModel.RtNode),
		filterMgr:               NewFilterMgr(FilterTypeMongo, conf),
	}
}

func (m *NodeMgr) SetConf(conf *config.Config) {
	m.conf = conf
	m.filterMgr.SetConf(conf)
}

func (m *NodeMgr) SetResources(resources resources.Resources) {
	m.resources = resources
	m.filterMgr.SetResources(resources)
}

func (m *NodeMgr) GetRootNodeByNodeId(nodeId string) *commonModel.RtNode {
	return m.allRootNodesMapByNodeId[nodeId]
}

func (m *NodeMgr) GetNodeByNodeId(nodeId string) *commonModel.RtNode {
	return m.allNodesMap[nodeId]
}

func (m *NodeMgr) GetNodeByIp() string {
	for _, node := range m.allNodesMap {
		for _, ipInfo := range node.Ips {
			if ipInfo.Ip == m.conf.Ip {
				_, ok := m.allRootNodesMapByNodeId[node.Id]
				fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "isRoot:", ok)
				return node.Id
			}
		}
	}
	return ""
}

type BwStatistics struct {
	avialiableNodeCnt  int
	avialiableIpCnt    int
	ispAvialiableBwMap map[string]float64
	ipParser           *ipdb.City
}

func (b *BwStatistics) OnNode(node *commonModel.RtNode) {
	//log.Println("node:", node.Id)
	b.avialiableNodeCnt++
}

func (b *BwStatistics) OnIp(node *commonModel.RtNode, ip *commonModel.RtIpStatus) {
	_, isp, _, _ := util.GetLocate(ip.Ip, b.ipParser)
	if ip.MaxInMBps > 0 && ip.OutMBps > 0 {
		b.avialiableIpCnt++
		b.ispAvialiableBwMap[isp] += (ip.MaxOutMBps - ip.OutMBps) * 8 / 1000
	}

}

func (b *BwStatistics) Done(result map[string]int) {
	fmt.Printf("BwStatistics Done, %+v\n", result)
}

func (m *NodeMgr) BwStatistics() {
	b := &BwStatistics{
		ispAvialiableBwMap: make(map[string]float64),
		ipParser:           m.resources.IpParser,
	}
	m.Register(b)
	m.filterMgr.SetFilterType(FilterTypeMongo)
	m.Traverse()
	log.Println("avialiableNodeCnt:", b.avialiableNodeCnt,
		"avialiableIpCnt", b.avialiableIpCnt)
	log.Println("ispAvialiableBwMap len:", len(b.ispAvialiableBwMap))
	for k, bw := range b.ispAvialiableBwMap {
		log.Printf("isp: %s, bw: %.1fGbps", k, bw)
	}
	m.filterMgr.SetFilterType(FilterTypeLocal)
	m.Traverse()
}

func (m *NodeMgr) LoadNodesFromFile(file string) bool {
	if _, err := os.Stat(file); err == nil {
		// 文件存在，从文件加载节点信息
		file, err := os.ReadFile(file)
		if err != nil {
			fmt.Println("LoadNodes ReadFile err:", err)
			return false
		}
		if err := json.Unmarshal(file, &m.allNodesMap); err != nil {
			fmt.Println("LoadNodes Unmarshal err:", err)
			return false
		}
		log.Println("从/tmp/allnodes.json文件加载节点信息成功", len(m.allNodesMap))
		return true
	}
	return false
}

const (
	allNodesCacheFile = "/tmp/allnodes.json"
	// livedRuntimeNodesAPI lived 运行时节点列表接口
	livedRuntimeNodesAPI = "http://miku-lived.qiniuapi.com:2240/v1/runtime/nodes"
	// livedNodesAPITimeout 节点列表接口超时，全量节点响应体较大，放宽到 15 分钟
	livedNodesAPITimeout = 15 * time.Minute
)

func (m *NodeMgr) LoadNodes() {
	log.Println("LoadNodes")
	// -from-redis=true 时保持原有逻辑：优先读 /tmp/allnodes.json，读不到再从 redis 拉取。
	// 默认为 false，改为从 lived 运行时接口拉取，成功后再写入 /tmp/allnodes.json 缓存。
	if m.conf != nil && m.conf.FromRedis {
		if m.LoadNodesFromFile(allNodesCacheFile) {
			return
		}
		m.loadNodesFromRedis()
		return
	}
	m.loadNodesFromHTTP()
}

func (m *NodeMgr) loadNodesFromRedis() {
	log.Println("redis:", m.resources.Redis)
	allNodes, err := commonModel.GetAllRTNodes(zerolog.Logger{}, m.resources.Redis)
	if err != nil {
		fmt.Println("LoadNodes GetAllRTNodes err:", err)
		return
	}
	m.setAllNodes(allNodes)
	// SetAllNodes 内部依赖 redis 读取 TCP 重传率，只有 redis 路径才能调用
	m.filterMgr.SetAllNodes(allNodes)
	log.Println("LoadNodes 成功")
}

// loadNodesFromHTTP 从 lived 运行时节点接口拉取全量节点，并缓存到 /tmp/allnodes.json
func (m *NodeMgr) loadNodesFromHTTP() {
	apiBase := livedRuntimeNodesAPI
	if m.conf != nil && strings.TrimSpace(m.conf.Addr) != "" {
		apiBase = strings.TrimSpace(m.conf.Addr)
	}
	log.Println("LoadNodes 从接口获取节点列表:", apiBase)

	client := &http.Client{Timeout: livedNodesAPITimeout}
	resp, err := client.Get(apiBase)
	if err != nil {
		log.Println("LoadNodes 请求接口失败:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("LoadNodes 读取响应失败:", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("LoadNodes 接口返回非 200: %d, body: %s\n", resp.StatusCode, truncateForLog(body))
		return
	}

	allNodes, err := parseRTNodes(body)
	if err != nil {
		log.Println("LoadNodes 解析节点列表失败:", err)
		return
	}
	if len(allNodes) == 0 {
		log.Println("LoadNodes 接口返回空节点列表")
		return
	}

	m.setAllNodes(allNodes)
	m.cacheNodesToFile()
	log.Printf("LoadNodes 成功, 节点数: %d\n", len(allNodes))
}

// parseRTNodes 兼容裸数组和 {code,data:{nodes:[...]}} / {data:[...]} 等包裹形式
func parseRTNodes(body []byte) ([]*commonModel.RtNode, error) {
	var direct []*commonModel.RtNode
	if err := json.Unmarshal(body, &direct); err == nil {
		return direct, nil
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, &direct); err == nil {
			return direct, nil
		}
		var wrapped struct {
			Nodes []*commonModel.RtNode `json:"nodes"`
			List  []*commonModel.RtNode `json:"list"`
		}
		if err := json.Unmarshal(envelope.Data, &wrapped); err == nil {
			if len(wrapped.Nodes) > 0 {
				return wrapped.Nodes, nil
			}
			if len(wrapped.List) > 0 {
				return wrapped.List, nil
			}
		}
	}
	return nil, fmt.Errorf("无法识别的节点列表格式: %s", truncateForLog(body))
}

func (m *NodeMgr) setAllNodes(allNodes []*commonModel.RtNode) {
	allNodesMap := make(map[string]*commonModel.RtNode, len(allNodes))
	for _, node := range allNodes {
		if node == nil {
			continue
		}
		allNodesMap[node.Id] = node
	}
	m.allNodesMap = allNodesMap
}

// cacheNodesToFile 把节点列表写入 /tmp/allnodes.json，下次启动可直接复用。
// 落盘格式与 DumpNodes/LoadNodesFromFile 保持一致（nodeId -> node 的 map）。
func (m *NodeMgr) cacheNodesToFile() {
	data, err := json.Marshal(m.allNodesMap)
	if err != nil {
		log.Println("cacheNodesToFile Marshal err:", err)
		return
	}
	if err := os.WriteFile(allNodesCacheFile, data, 0644); err != nil {
		log.Println("cacheNodesToFile WriteFile err:", err)
		return
	}
	log.Println("节点列表已缓存到", allNodesCacheFile, "len:", len(m.allNodesMap))
}

func truncateForLog(body []byte) string {
	const limit = 500
	s := strings.TrimSpace(string(body))
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "...(truncated)"
}

func (m *NodeMgr) DumpNodes() {
	data, err := json.Marshal(m.allNodesMap)
	if err != nil {
		fmt.Println("DumpNodes Marshal err:", err)
		return
	}
	filename := fmt.Sprintf("nodes-%d.json", time.Now().Unix())
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Println("DumpNodes WriteFile err:", err)
		return
	}

	util.UploadFile(filename)
	fmt.Println("DumpNodes 成功")
}

func (m *NodeMgr) WriteNodesToRedis() {
	if !m.LoadNodesFromFile("/tmp/allnodes.json") {
		return
	}
	log.Println("len:", len(m.allNodesMap))
	for nodeId, node := range m.allNodesMap {
		nodeData, err := json.Marshal(node)
		if err != nil {
			log.Println("WriteNodesToRedis Marshal err:", err, "key:", nodeId)
			continue
		}
		_, err = m.resources.Redis.HSet(context.Background(), "mik_netprobe_runtime_nodes_map",
			nodeId, string(nodeData)).Result()
		if err != nil {
			log.Println("WriteNodesToRedis HSet err:", err, "key:", nodeId)
		}
	}
}

func (m *NodeMgr) dumpOutProvNodes() {
	m.filterMgr.LoadFilterData()
	allNodes := m.allNodesMap
	statusCntMap := make(map[string]int)
	for _, node := range allNodes {
		if !node.IsDynamic {
			continue
		}
		statusCntMap["total"]++
		nodeAbilility := m.filterMgr.GetNodeAbility(node.Id)
		if nodeAbilility == nil {
			log.Println("dumpOutProvNodes GetNodeAbility err:", node.Id)
			continue
		}

	}
}

var douyulist = []string{
	"ff20aba243c144fe1dd15202cbb54675",
	"04dc7a3f19e0e1aba5a1015407c48d4e",
	"4b57785dcad347d5ea3497f3c85a6c72",
	"b72c51ed7a4f1a38836037d6d95cc9d8",
	"6e9c0ed96c4cc968728d97727ae2eb17",
	"4b3ef6efdbeaf3b53726487737e85c81",
	"643f5908a79ab5fe370cc4c270f1915a",
	"b313ef6513801afbaf9db826538a10ad",
	"b72c51ed7a4f1a38836037d6d95cc9d8",
	"380477484dda70ca56f878dac9d26dc6",
	"38d88cffd2c0a9c3d8abf09fd0a28886",
}

func (m *NodeMgr) Filternode() {
	file, err := os.ReadFile("/tmp/nodelist.txt")
	if err != nil {
		log.Println("Read nodelist.txt err:", err)
		return
	}
	lines := strings.Split(string(file), "\n")
	machidMap := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		machidMap[line] = true
	}
	log.Println("len(machidMap)", len(machidMap))
	log.Println("len(m.allNodesMap)", len(m.allNodesMap))
	cnt := 0
	for _, node := range m.allNodesMap {
		if !node.IsDynamic {
			continue
		}
		if !machidMap[node.MachineId] {
			continue
		}

		_, _, area, province := util.GetNodeLocate(node, m.resources.IpParser)
		if util.ContainInStringSlice(douyulist, node.MachineId) {
			log.Println("skip", node.MachineId, area, province)
			continue
		}

		if area == "华东" || area == "华中" /*|| province == "黑龙江"*/ {
			fmt.Println(node.Id, node.MachineId, area, province)
			cnt++
		}
	}
	log.Println("cnt", cnt)
}

type Response struct {
	NodeStatus302
}

// NodeStatus 表示节点状态信息
type NodeStatus302 struct {
	MikuNodeid string `json:"mikuNodeid"`
	ID         string `json:"id"`
	CreateAt   string `json:"createAt"`
	UpdateAt   string `json:"updateAt"`
	NodeID     string `json:"nodeId"`
	NodeID32   int    `json:"nodeId32"`
	MachineId  string `json:"host"`
	Status     string `json:"status"`
	BwMbps     int    `json:"bwMbps"`
	//CPU                      CPUInfo                `json:"cpu"`
	//Memory                   MemoryInfo             `json:"memory"`
	Uptime int `json:"uptime"`
	Date   int `json:"date"`
	//Services                 map[string]ServiceInfo `json:"services"`
	//Disks                    map[string]DiskInfo    `json:"disks"`
	//DiskIOStats              []DiskIOStat           `json:"diskIOStats"`
	RebootStatus   int     `json:"rebootStatus"`
	TCPRetransRate float64 `json:"tcpRetransRate"`
	MovingAvg      float64 `json:"movingAvg"`
	Abandoned      bool    `json:"abandoned"`
	HTTPOnly       bool    `json:"httpOnly"`
	ResourceType   string  `json:"resourceType"`
	//Ips                      []IPStatus `json:"ips"`
	Quality                  string   `json:"quality"`
	QualityTags              []int    `json:"qualityTags"`
	Area                     string   `json:"area"`
	ManualArea               string   `json:"manualArea"`
	ISP                      string   `json:"isp"`
	ForbidMiku               bool     `json:"forbidMiku"`
	ScheduleArea             []string `json:"scheduleArea"`             // 可调度区域: 优先级高, 人工修改
	NiulinkScheduleArea      []string `json:"niulinkScheduleArea"`      // 可调度区域
	AnalyzerBlacklistedAreas []string `json:"analyzerBlacklistedAreas"` // 黑名单区域
	NiulinkBlacklistedAreas  []string `json:"niulinkBlacklistedAreas"`  // 人工黑名单区域
}

func (m *NodeMgr) DownloadNodeStatus(conf *config.Config) {
	if conf.Node == "" {
		log.Println("DownloadNodeStatus node is empty")
		return
	}
	start_time := time.Now()
	logger := zlog.Logger.With().Str("DownloadNodeStatus machineId", conf.Node).Logger()

	// http://dn_scheduler.fusion.internal.qiniu.io/dn/node?id=vdnd-qnvm-<machineId>
	baseURL := "http://dn_scheduler.fusion.internal.qiniu.io/dn/node"
	params := url.Values{}
	params.Add("id", "vdnd-qnvm-"+conf.Node)

	client := &http.Client{}
	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		logger.Error().Err(err).Msg("DownloadNodeStatus Failed")
		return
	}
	defer func() {
		if time.Since(start_time).Milliseconds() > 1000 {
			logger.Info().Msgf(" end, elapsed_time: %v", time.Since(start_time))
		}
	}()

	resp, err := client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msg("请求失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error().Int("status_code", resp.StatusCode).Str("url", reqURL).Msg("请求失败")
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("读取响应体失败")
		return
	}

	var responseData Response
	if err := json.Unmarshal(body, &responseData); err != nil {
		logger.Error().Err(err).Msgf("JSON 解析失败")
		return
	}
	data, err := json.MarshalIndent(responseData.NodeStatus302, "", "  ")
	if err != nil {
		logger.Error().Err(err).Msgf("JSON 序列化失败")
		return
	}
	fmt.Println(string(data))
}

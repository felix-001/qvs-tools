package staging

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	deflog "log"
	"math"
	"middle-source-analysis/nodetraverse"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	localUtil "middle-source-analysis/util"

	localPublic "middle-source-analysis/public"

	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/cmd/sched/model"
	"github.com/qbox/mikud-live/common"
	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/rs/zerolog/log"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/exp/rand"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 临时的代码放到这里

func Staging() {
	switch conf.SubCmd {
	case "getpcdn":
		getPcdnFromSchedAPI(true, false)
	case "volc":
		fetchVolcOriginUrl()
	case "ipv6":
		dumpNodeIpv4v6Dis()
	case "hashRingMap":
		HashRingMap()
	case "refactor":
		Refactor()
	case "load":
		Load()
	case "exec":
		Exec()
	case "dns":
		Dnspod()
	case "log":
		m := make(map[string]string)
		m["aaa"] = "hello"
		m["bbb"] = "world"
		m["ccc"] = "foo"
		logger.Info().Any("mm", m).Msg("test")
	case "lowbw":
		//LowBw()
		//buildAllNodesMap()
		DumpNodes()
	case "dnsrecords":
		DnsRecords()
	case "lowbw2":
		buildAllNodesMap()
		LowBw2()
	case "deldns":
		DelDns()
	case "lines":
		DnsLines()
	case "nodes":
		GenNodes()
	case "dump":
		dumpNodes2()
	case "dumpNodeFromFile":
		dumpNodeFromFile()
	case "retrans":
		Retrans()
	case "80port":
		Port80()
	case "xml":
		Xml()
	case "nat1":
		Nat1()
	case "watermark":
		Watermark()
	case "font":
		Font()
	case "angle":
		TestAngle()
	case "sips":
		AllSipService()
	case "qps":
		Qps()
	case "sipraw":
		SipRaw()
	case "talk":
		TalkTest()
	case "uac":
		uac("udp")
		uac("tcp")
		time.Sleep(10 * time.Second)
	case "dev":
		getDevices()
	case "rtc":
		rtcMemLeakTest()
	case "rtptest":
		rtpTest()
	case "res":
		//nodetraverse.RegisterMultiIPChk()
		//nodetraverse.Traverse(conf.RedisAddrs, conf.IPDB, s)
	case "report":
		ReportChk()
	}
}

type NodeDis struct {
	Ipv4Cnt int
	Ipv6Cnt int
}

func dumpNodeIpv4v6Dis() {
	areaIpv4v6CntMap := make(map[string]*NodeDis)
	for _, node := range allNodesMap {
		for _, ip := range node.Ips {
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			if !localUtil.IsPublicIPAddress(ip.Ip) {
				continue
			}
			isp, area, _ := localUtil.GetLocate(ip.Ip, IpParser)
			if area == "" {
				continue
			}
			key := area + "_" + isp
			m := areaIpv4v6CntMap[key]
			if m == nil {
				m = &NodeDis{}
				areaIpv4v6CntMap[key] = m
			}
			if ip.IsIPv6 {
				m.Ipv6Cnt++
			} else {
				m.Ipv4Cnt++
			}
		}
	}

	areaIpv6PercentMap := make(map[string]int)
	for areaIsp, nodeDis := range areaIpv4v6CntMap {
		areaIpv6PercentMap[areaIsp] = nodeDiIpv6Cnt * 100 / (nodeDiIpv4Cnt + nodeDiIpv6Cnt)
	}
	pairs := localUtil.SortIntMap(areaIpv6PercentMap)
	localUtil.DumpSlice(pairs)

}

const (
	// DefaultVirtualSpots default virtual spots
	DefaultVirtualSpots = 10
	HashRingLength      = 4096
)

type ringNode struct {
	nodeKey   string
	spotValue uint32
}

type nodesArray []ringNode

func (p nodesArray) Len() int           { return len(p) }
func (p nodesArray) Less(i, j int) bool { return p[i].spotValue < p[j].spotValue }
func (p nodesArray) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p nodesArray) Sort()              { sort.Sort(p) }

// HashRing store nodes
type HashRing struct {
	Algorithm string

	virtualSpots int
	nodes        nodesArray
	originNodes  map[string]model.NodeIpsPair

	rawPairList  []model.NodeIpsPair // 原始数据
	sortType     int                 // 带宽排序类型，remainRatio or remainBw
	totalWeight  int                 // 总权重
	weightList   []int               // 权重列表
	preSumWeight []int               // (权重)前缀和，实际使用

	Random *rand.Rand
	mu     sync.RWMutex
}

// NewHashRing create a hash ring with virtual spots
func NewHashRing(spots, sortType int, algorithm string) *HashRing {
	if spots == 0 {
		spots = DefaultVirtualSpots
	}

	h := &HashRing{
		Algorithm: algorithm,

		virtualSpots: spots,
		originNodes:  make(map[string]model.NodeIpsPair),

		rawPairList:  make([]model.NodeIpsPair, 0),
		sortType:     sortType,
		totalWeight:  0,
		weightList:   make([]int, 0),
		preSumWeight: make([]int, 0),

		Random: rand.New(rand.NewSource(uint64(time.Now().UnixNano()))),
	}
	return h
}

// AddNodes add nodes to hash ring
func (h *HashRing) AddNodes(originNodes []model.NodeIpsPair) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, pair := range originNodes {
		if pair.Node == nil || len(pair.Ips) == 0 {
			continue
		}

		h.originNodes[pair.Node.Id] = originNodes[i]
	}

	h.rawPairList = append(h.rawPairList, originNodes...)
	sortPairList(h.rawPairList, h.sortType)
	switch h.sortType {
	case constBwSortByRemainRatio:
		weightList := make([]int, 0)
		for _, pair := range h.rawPairList {
			ratio2Weight := int(pair.RemainRatio * 100)
			h.totalWeight += ratio2Weight
			weightList = append(weightList, ratio2Weight)
		}
		h.weightList = weightList
		h.preSumWeight = calPreSum(weightList)
	//case constBwSortByRemainBw:
	default:
		weightList := make([]int, 0)
		for _, pair := range h.rawPairList {
			bw2Weight := int(pair.RemainMBps / 10)
			h.totalWeight += bw2Weight
			weightList = append(weightList, bw2Weight)
		}
		h.weightList = weightList
		h.preSumWeight = calPreSum(weightList)
	}

	if h.totalWeight <= 0 {
		//msg := fmt.Sprintf("Invalid totalWeight:%d", h.totalWeight)
		//go alarm.SendWeChatAlarm(constWechatAlarmUrl, "HashRing", "", msg, constErrInvalidTotalWeight)

		h.totalWeight = 1
	}

	h.generate()
}

// calPreSum 计算前缀和
func calPreSum(weightList []int) []int {
	preSum := make([]int, 0)
	for i := 0; i < len(weightList); i++ {
		if i == 0 {
			preSum = append(preSum, weightList[0])
			continue
		}

		preSum = append(preSum, preSum[i-1]+weightList[i])
	}

	return preSum
}

// AddNode add node to hash ring
func (h *HashRing) AddNode(node model.NodeIpsPair) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if node.Node == nil || len(node.Ips) == 0 {
		return
	}

	h.originNodes[node.Node.Id] = node
	h.generate()
}

// RemoveNode remove node
func (h *HashRing) RemoveNode(nodeKey string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.originNodes, nodeKey)
	h.generate()
}

func (h *HashRing) generate() {
	h.nodes = nodesArray{}
	for nodeKey := range h.originNodes {
		for i := 1; i <= h.virtualSpots; i++ {
			hashVal := util.FnvHash(nodeKey+":"+strconv.Itoa(i)) % HashRingLength
			n := ringNode{
				nodeKey:   nodeKey,
				spotValue: hashVal,
			}
			h.nodes = append(h.nodes, n)
		}
	}
	h.nodeSort()
}

// GetNodeKey get node by streamId
func (h *HashRing) GetNodeKey(streamId string) string {
	if len(h.nodes) == 0 {
		return ""
	}

	hashVal := util.FnvHash(streamId) % HashRingLength
	i := sort.Search(len(h.nodes), func(i int) bool { return h.nodes[i].spotValue >= hashVal })

	if i >= len(h.nodes) {
		i = 0
	}
	return h.nodes[i].nodeKey
}

func (h *HashRing) GetNode(streamId string) *model.NodeIpsPair {
	switch h.Algorithm {
	case constAlgorithmConsistentHash:
		return h.GetNodeByConsistentHash(streamId)
	case constAlgorithmBwShare:
		return h.GetNodeByBwFairShare()
	default:
		log.Error().Msgf("HashRing invalid algorithm:%s", h.Algorithm)
		return h.GetNodeByConsistentHash(streamId)
	}
}

// GetNodeByBwFairShare 带宽公平份额
func (h *HashRing) GetNodeByBwFairShare() *model.NodeIpsPair {
	if h.totalWeight <= 0 {
		log.Error().Msgf("GetNodeByBwFairShare invalid totalWeight:%d", h.totalWeight)
		h.totalWeight = 1
	}

	random := h.Random.Intn(h.totalWeight) + 1
	idx := sort.SearchInts(h.preSumWeight, random)

	if idx >= 0 && idx < len(h.rawPairList) {
		return &h.rawPairList[idx]
	}

	if idx >= len(h.rawPairList) {
		log.Error().Msgf("GetNodeByBwFairShare invalid idx:%d, len(h.rawPairList):%d", idx, len(h.rawPairList))
	}

	return nil
}

// GetNodeByConsistentHash 一致性hash
func (h *HashRing) GetNodeByConsistentHash(streamId string) *model.NodeIpsPair {
	nodeKey := h.GetNodeKey(streamId)
	if nodeKey == "" {
		return nil
	}

	if pair, ok := h.originNodes[nodeKey]; ok {
		return &pair
	}

	return nil
}

func (h *HashRing) GetOriginNodes() map[string]model.NodeIpsPair {
	return h.originNodes
}

func buildRingGroup(groups map[string][]model.NodeIpsPair, sortType int, algorithm string) map[string]*HashRing {
	res := make(map[string]*HashRing)
	for groupName, pairList := range groups {
		ring := NewHashRing(DefaultVirtualSpots, sortType, algorithm)
		ring.AddNodes(pairList)

		res[groupName] = ring
	}

	return res
}

func sortPairList(pairList []model.NodeIpsPair, sortType int) {
	for i, pair := range pairList {
		var nodeMaxBw float64 = 0
		var nodeRemainBw float64 = 0
		for j, ipInfo := range pair.Ips {
			// ip remainBw
			remainBw := ipInfo.MaxOutMBps - ipInfo.OutMBps
			if remainBw <= 0 {
				remainBw = 0
			}
			pairList[i].Ips[j].RemainOutMBps = remainBw

			// ip remainRatio
			var remainRatio float64 = 0
			if ipInfo.MaxOutMBps > 0 && remainBw > 0 {
				remainRatio = remainBw / ipInfo.MaxOutMBps
			}
			pairList[i].Ips[j].RemainOutBwRatio = remainRatio

			nodeMaxBw += ipInfo.MaxOutMBps
			nodeRemainBw += remainBw
		}

		var nodeRemainRatio float64 = 0
		if nodeMaxBw > 0 && nodeRemainBw > 0 {
			nodeRemainRatio = nodeRemainBw / nodeMaxBw
		}
		pairList[i].RemainRatio = nodeRemainRatio // node remainRatio
		pairList[i].RemainMBps = nodeRemainBw     // node remainBw

		if sortType == constBwSortByRemainRatio {
			sort.Sort(model.IpListByRemainRatio(pairList[i].Ips))
		} else {
			sort.Sort(model.IpListByRemainBw(pairList[i].Ips))
		}
	}

	if sortType == constBwSortByRemainRatio {
		sort.Sort(model.PairListByRemainRatio(pairList))
	} else {
		sort.Sort(model.PairListByRemainBw(pairList))
	}
}

func HashRingMap() {
	/*
		levels := []string{"default", "level1", "level2"}
		ipVersions := []string{"ipv4", "ipv6", "all"}
		isps := []string{"移动", "电信", "联通"}
		areas := Areas
		provinces := Provinces
		outProvince := []bool{true, false}
		countries := []string{"中国", "海外"}

		count := 0
		for _ := range levels {
			for _, _ := range ipVersions {
				for _, _ := range isps {
					for _, _ := range areas {
						for _, _ := range provinces {
							for _, _ := range councountries {
								count++
							}
						}
					}
				}
			}
		}
	*/
	var originNodes []model.NodeIpsPair
	for i := 0; i < 10000; i++ {
		nodeIpPair := model.NodeIpsPair{
			Node: &public.RtNode{
				Node: public.Node{
					Id:     fmt.Sprintf("node%d", i),
					Status: "normal",
				},
			},
		}
		originNodes = append(originNodes, nodeIpPair)
	}
	start := time.Now()
	for i := 0; i < 5000; i++ {
		ring := NewHashRing(DefaultVirtualSpots, 2, "hash")
		ring.AddNodes(originNodes)
	}
	fmt.Println("cost", time.Since(start))
}

/*
不出省
省内
大区
isp
跨isp
*/

type NodeResourceRatio struct {
	Ratio  map[string]float64 `json:"ratio"` // 边缘分发节点的资源配比
	Backup *NodeResourceRatio `json:"backup"`
	Relay  *NodeResourceRatio `json:"relay"`
	//Relay map[string]map[string]float64 `json:"relay"` // relay节点资源配比, key1: 层级 key2: 资源类型
}

type ResourceConfig struct {
	NodesQualityLevel common.NodesQualityLevel `json:"nodes_quality_level"` // 节点质量等级,default/level1...
	FallbackIspKey    string                   `json:"fallback_isp_key"`    // 兜底isp
	CrossAreaEnable   bool                     `json:"cross_area_enable"`   // 是否允许跨区选点
	CrossIspEnable    bool                     `json:"cross_isp_enable"`    // 是否允许跨运营商选点
	Algorithm         string                   `json:"algorithm"`           // 选点算法, hash/bwShare
	MaxNodes          int                      `json:"max_nodes"`           // 热流分散选点最大值
	QpmWindowSize     int64                    `json:"qpm_window_size"`     // 统计qpm滑动窗口大小
	QpmGroupInterval  int64                    `json:"qpm_group_interval"`  // 统计qpm滑动窗口每一个slot大小
	HotThresold       int                      `json:"hot_thresold"`        // 热流qpm阈值
	ResourceRatio     NodeResourceRatio        `json:"resource_ratio"`      // 资源配比
	Ipv6Ratio         float64                  `json:"ipv6_ratio"`          // ipv6比例
	Enable302         bool                     `json:"enable_302"`          // 是否支持高负载302调走
}

func Refactor() {
	// 配置节点质量等级为default, 选点算法为"hash", 50%专线, 50%汇聚, 不使用relay
	//cfg := ResourceConfig{}

	cfg := NodeResourceRatio{
		Ratio: map[string]float64{
			"盒子": 30,
			"汇聚": 70,
		},
		Relay: &NodeResourceRatio{
			Ratio: map[string]float64{
				"专线root": 100,
			},
			Relay: &NodeResourceRatio{
				Ratio: map[string]float64{
					"专线root": 100,
				},
			},
		},
		Backup: &NodeResourceRatio{
			Ratio: map[string]float64{
				"专线": 100,
			},
		},
	}
	jsonbody, err := json.Marshal(&cfg)
	if err != nil {
		log.Info().Err(err).Msg("")
		return
	}
	fmt.Println(string(jsonbody))
}

func Load() {
	cnt := 0
	total := 0
	nodes := make(map[string]string)
	for _, node := range allNodesMap {
		if node.RuntimeStatus != "Serving" {
			continue
		}
		if !util.CheckNodeUsable(zlog.Logger, node, constTypeLive) {
			continue
		}

		if node.StreamdPortHttp <= 0 || node.StreamdPortHttps <= 0 || node.StreamdPortWt <= 0 {
			continue
		}
		for _, ipInfo := range node.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			if ipInfo.IPStreamProbe.State != public.StreamProbeStateSuccess {
				continue
			}
			if ipInfo.IPStreamProbe.Speed < 12 && ipInfo.IPStreamProbe.MinSpeed < 10 {
				continue
			}
			usage := (ipInfo.OutMBps * float64(100)) / ipInfo.MaxOutMBps
			if usage >= 93 {
				cnt++
				nodes[node.Id] = ipInfo.Ip
			}
			total++
		}
	}
	logger.Info().Int("cnt", cnt).Int("total", total).Msg("")
	for nodeid, ip := range nodes {
		logger.Info().Str("node", nodeid).Str("ip", ip).Msg("")
	}
}

func Exec() {
	cmd := exec.Command("jumpboxCmdNew", "redis-cli -h 10.70.60.31 -p 8200 -c --raw hgetall mik_netprobe_runtime_nodes_map")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println("the output is:", string(output))
}

func Dnspod() {
	cli, err := tencent_dnspod.NewTencentClient(conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	/*
		op := public.Operation{
			Type:  "A",
			Line:  "默认",
			Value: "119.145.128.120",
		}
		xl := xlog.NewDummyWithCtx(context.Background())
			resp1, err := cli.CreateRaw(xl, &op, "zeicaefiegoh.com", "*")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(resp1)
			op = public.Operation{
				Type:  "A",
				Line:  "默认",
				Value: "119.145.128.198",
			}
			resp1, err = cli.CreateRaw(xl, &op, "zeicaefiegoh.com", "bbbbbb")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(resp1)
			op = public.Operation{
				Type:  "A",
				Line:  "默认",
				Value: "119.145.128.197",
			}
			resp1, err = cli.CreateRaw(xl, &op, "zeicaefiegoh.com", "ccccc")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(resp1)
	*/

	//fmt.Println(cli)
	//resp, err := cli.GetLines(xl, "zeicaefiegoh.com", "")
	resp, err := cli.GetLines(xl, "zeicaefiegoh.com", "")
	//resp, err := cli.GetLines(xl, "mikudncom", "")
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(resp)
	for k, v := range resp {
		fmt.Println(k)
		fmt.Println("\tname:", *v.Name, "id:", *v.LineId)
	}
	//resp1, err := cli.GetDomain(xl, "zeicaefiegoh.com")
	resp1, err := cli.GetDomain(xl, "zeicaefiegoh.com")
	if err != nil {
		fmt.Println(err)
		return
	}

	bytes, err := json.MarshalIndent(resp1, "", "  ")
	if err != nil {
		return
	}
	fmt.Println(string(bytes))

	/*
		resp, err := cli.GetRecords(xl, "cloudvdn.com", "2006024383", "", 0)
		if err != nil {
			fmt.Println(err)
			return
		}
		//fmt.Println(resp)
		fmt.Println(*resp.RecordCountInfo.ListCount)
		fmt.Println(*resp.RecordCountInfo.SubdomainCount)
		fmt.Println(*resp.RecordCountInfo.TotalCount)
		//fmt.Println(resp.RecordList)
		areaIpsMap := make(map[string][]string)
		for i, record := range resp.RecordList {
			fmt.Println()
			fmt.Println(i)
			fmt.Println("defaultns", *record.DefaultNS)
			fmt.Println("value", *record.Value)
			fmt.Println("name", *record.Name)
			fmt.Println("line", *record.Line)
			fmt.Println(*record.Type)
			//fmt.Println(*record.Weight)
			fmt.Println("remark", *record.Remark)
			fmt.Println("ttl", *record.TTL)
			areaIpsMap[*record.Line] = append(areaIpsMap[*record.Line], *record.Value)
		}

		fmt.Println(areaIpsMap)
		bytes, err := json.MarshalIndent(areaIpsMap, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(bytes))
	*/
	/*
		op := public.Operation{
			Type:  "A",
			Line:  "默认",
			Value: "119.145.128.191",
		}
		resp1, err := cli.CreateRaw(xl, &op, "qnrd.volclivedvcom", "abc123hello")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(resp1)

		op1 := public.Operation{
			Type:  "A",
			Line:  "默认",
			Value: "218.22.23.189",
		}
		resp1, err = cli.CreateRaw(xl, &op1, "qnrd.volclivedvcom", "abc123hello")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(resp1)
	*/

}

func getAllNodes() string {
	raw := "redis-cli -h 10.70.60.31 -p 8200 -c --raw hgetall mik_netprobe_runtime_nodes_map"
	cmd := exec.Command("jumpboxCmdNew", raw)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return ""
	}
	//fmt.Println("the output is:", string(output))
	pos := stringIndex(string(output), raw)
	if pos == -1 {
		fmt.Println("find mik_netprobe_runtime_nodes_map && exti err")
		return ""
	}
	pos += len(raw) + 2
	str := string(output)[pos:]

	pos = stringIndex(str, raw)
	if pos == -1 {
		fmt.Println("find mik_netprobe_runtime_nodes_map && exti err")
		return ""
	}
	pos += len(raw) + 10
	return string(str)[pos:]
}

func LowBw() {
	file := "/tmp/nodejson"
	_, err := oStat(file)

	output := ""
	if oIsNotExist(err) || conf.Force {
		output = getAllNodes()
		err := ioutil.WriteFile(file, []byte(output), 0644)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else if err != nil {
		fmt.Printf("检查文件 %s 时发生错误: %s\n", file, err)
		return
	} else {
		logger.Info().Str("file", file).Msg("read from file")
		bytes, err := ioutil.ReadFile(file)
		if err != nil {
			logger.Error().Err(err).Str("file", file).Msg("read file err")
			return
		}
		output = string(bytes)
	}
	if output == "" {
		return
	}

	//fmt.Println("output:", output)
	allNodesMap := make(map[string]*public.RtNode)
	lines := strings.Split(output, "\r\n")
	fmt.Println("lines:", len(lines))
	for i, line := range lines {
		if i%2 == 0 {
			continue
		}
		node := &public.RtNode{}
		if err := json.Unmarshal([]byte(line), node); err != nil {
			logger.Error().Err(err).Int("i", i).Msg("unmarshal")
			continue
		}
		allNodesMap[node.Id] = node
	}
	fmt.Println("nodes count:", len(allNodesMap))
	ipCnt := 0
	dedicatedCnt := 0
	aggregationCnt := 0
	dedicatedIpCnt := 0
	aggregationIpCnt := 0
	more32IpCnt := 0
	nat1Cnt := 0
	nat1IpCnt := 0
	more20IpCnt := 0
	notServingCnt := 0
	notNormalCnt := 0
	servingCnt := 0
	for _, node := range allNodesMap {
		if !node.IsDynamic {
			continue
		}
		if node.IsNat1() {
			nat1Cnt++
			//continue
		}
		if node.Status != "Normal" {
			notNormalCnt++
		}
		if node.RuntimeStatus != "Serving" {
			notServingCnt++
			continue
		}
		if len(node.Ips) == 0 {
			continue
		}
		ability, ok := node.Abilities["live"]
		if !ok || !ability.Can || ability.Frozen {
			logger.Warn().Msgf("[CheckNodeUsable] not pass, nodeId:%s, machineId:%s, isDynamic:%t, abilities:%+v",
				node.Id, node.MachineId, node.IsDynamic, node.Abilities)
			continue
		}

		if _, ok = node.Services["live"]; !ok {
			logger.Warn().Msgf("[CheckNodeUsable] not pass, nodeId:%s, machineId:%s, isDynamic:%t, services:%+v",
				node.Id, node.MachineId, node.IsDynamic, node.Services)
			continue
		}
		if node.ResourceType == "dedicated" {
			dedicatedCnt++
		} else if node.ResourceType == "aggregation" {
			aggregationCnt++
		}
		for _, ipInfo := range node.Ips {
			servingCnt++
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			if node.ResourceType == "dedicated" {
				dedicatedIpCnt++
			} else if node.ResourceType == "aggregation" {
				aggregationIpCnt++
			}
			if ipInfo.MaxOutMBps > 32 {
				more32IpCnt++
			}
			if node.IsNat1() {
				nat1IpCnt++
			}
			if ipInfo.MaxOutMBps > 20 {
				more20IpCnt++
			}
			ipCnt++
		}
	}
	logger.Info().
		Int("ipCnt", ipCnt).
		Int("dedicatedCnt", dedicatedCnt).
		Int("aggregationCnt", aggregationCnt).
		Int("dedicatedIpCnt", dedicatedIpCnt).
		Int("aggregationIpCnt", aggregationIpCnt).
		Int("more32IpCnt", more32IpCnt).
		Int("nat1Cnt", nat1Cnt).
		Int("nat1IpCnt", nat1IpCnt).
		Int("more20IpCnt", more20IpCnt).
		Int("notServingCnt", notServingCnt).
		Int("notNormalCnt", notNormalCnt).
		Int("servingCnt", servingCnt).
		Msg("LowBw")
}

func isProvinceLine(line string) bool {
	for _, province := range localPublic.Provinces {
		if stringContains(line, province) {
			return true
		}
	}
	return false
}

func isAreaLine(line string) bool {
	for _, area := range localPublic.Areas {
		if stringContains(line, area) {
			return true
		}
	}
	return false
}

func DnsRecords() {
	cli, err := tencent_dnspod.NewTencentClient(conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := cli.GetRecords(xl, conf.Domain, "", "", 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	/*
		bytes, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
			err = oWriteFile("/tmp/out.log", bytes, 0644)
			if err != nil {
				fmt.Println(err)
				return
			}
	*/
	lineMap := make(map[string]map[string][]string)

	totalProvV4Cnt := 0
	totalProvV6Cnt := 0

	totalAreaV4Cnt := 0
	totalAreaV6Cnt := 0

	//areaMap := make(map[string]map[string][]string)
	for _, record := range resp.RecordList {
		if record.Name == nil {
			continue
		}
		if *record.Name != conf.Name {
			continue
		}
		if record.Line == nil {
			continue
		}
		if record.Value == nil {
			continue
		}
		if record.Type == nil {
			continue
		}
		if _, ok := lineMap[*record.Line]; !ok {
			lineMap[*record.Line] = make(map[string][]string)
		}
		logger.Info().Str("line", *record.Line).Str("name", *record.Name).Str("status", *record.Status).
			Str("value", *record.Value).Uint64("recordid", *record.RecordId).Msg("")
		lineMap[*record.Line][*record.Type] = append(lineMap[*record.Line][*record.Type], *record.Value)

		if isProvinceLine(*record.Line) {
			if *record.Type == "A" {
				totalProvV4Cnt++
			}
			if *record.Type == "AAAA" {
				totalProvV6Cnt++
			}
		}

		if isAreaLine(*record.Line) {
			if *record.Type == "A" {
				totalAreaV4Cnt++
			}
			if *record.Type == "AAAA" {
				totalAreaV6Cnt++
			}
		}

		/*
			isp := ""
			prov := ""
			for _, isp = range Isps {
				if stringContains(*record.Line, isp) {
					prov = stringReplaceAll(*record.Line, isp, "")
					break
				}
			}
			if isp == "" || prov == "" {
				logger.Error().Str("line", *record.Line).Msg("DnsRecords")
				continue
			}
			area, _ := schedUtil.ProvinceAreaRelation(prov)
			if util.ContainInStringSlice(prov, Areas) {
				area = prov
			}
			key := area + isp
			if _, ok := areaMap[key]; !ok {
				areaMap[key] = make(map[string][]string)
			}
			areaMap[key][*record.Type] = append(areaMap[key][*record.Type], *record.Value)
		*/
	}
	for line, typeMap := range lineMap {
		fmt.Printf("%s:\n", line)
		for t, ips := range typeMap {
			fmt.Printf("\t%s:\n", t)
			fmt.Printf("\t\t%d %+v\n", len(ips), ips)
		}
	}
	fmt.Println("totalProvV4Cnt:", totalProvV4Cnt)
	fmt.Println("totalProvV6Cnt:", totalProvV6Cnt)
	fmt.Println("totalAreaV4Cnt:", totalAreaV4Cnt)
	fmt.Println("totalAreaV6Cnt:", totalAreaV6Cnt)

	/*
		redisCli := rediNewClusterClient(&rediClusterOptions{
			Addrs:      conf.RedisAddrs,
			MaxRetries: 3,
			PoolSize:   30,
		})

		err = redisCli.Ping(context.Background()).Err()
		if err != nil {
			fmt.Println(err)
			return
		}

		allNodes, err := public.GetAllRTNodes(logger, redisCli)
		if err != nil {
			logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
			return
		}
		for areaIsp, typeMap := range areaMap {
			fmt.Printf("%s:\n", areaIsp)
			for t, ips := range typeMap {
				fmt.Printf("\t%s:\n", t)
				fmt.Printf("\t\t%d %+v\n", len(ips), ips)
				bw := calcBw(allNodes, ips)
				fmt.Printf("\t\t%.1f Gbps\n", bw*8/1000)
			}
		}

		fmt.Printf("len: %d 大区覆盖率: %.0f%%\n", len(areaMap), float64(len(areaMap))*100.0/float64(21))

		needAreas := []string{}
		for _, area := range Areas {
			for _, isp := range Isps {
				areaIsp := area + isp
				if _, ok := areaMap[areaIsp]; !ok {
					needAreas = append(needAreas, areaIsp)
				}
			}
		}
		fmt.Println("没有覆盖的大区:", needAreas)
	*/
}

func calcBw(allNodes []*public.RtNode, ips []string) (totalBw float64) {
	for _, node := range allNodes {
		for _, ipInfo := range node.Ips {
			for _, ip := range ips {
				if ip != ipInfo.Ip {
					continue
				}
				totalBw += ipInfo.MaxOutMBps
			}
		}
	}
	return
}

func redisGetAllNodes() []*public.RtNode {
	redisCli := rediNewClusterClient(&rediClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		zlog.Fatal().Err(err).Msg("")
	}

	allNodes, err := public.GetAllRTNodes(logger, redisCli)
	if err != nil {
		logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return nil
	}
	return allNodes
}

func DumpNodes() {
	redisCli := rediNewClusterClient(&rediClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		zlog.Fatal().Err(err).Msg("")
	}

	allNodes, err := public.GetAllRTNodes(logger, redisCli)
	if err != nil {
		logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	nodeMap := make(map[string]int)
	for _, node := range allNodes {
		if !node.IsDynamic {
			nodeMap["staicNode"]++
			continue
		}
		nodeMap["afterSetp1"]++
		if node.RuntimeStatus != "Serving" {
			nodeMap["notServing"]++
			continue
		}
		nodeMap["afterSetp2"]++
		if len(node.Ips) == 0 {
			nodeMap["noIps"]++
			continue
		}
		nodeMap["afterSetp3"]++
		ability, ok := node.Abilities["live"]
		if !ok || !ability.Can || ability.Frozen {
			nodeMap["abilityChk"]++
			continue
		}
		nodeMap["afterSetp4"]++

		if _, ok = node.Services["live"]; !ok {
			nodeMap["servicesChk"]++
			continue
		}
		nodeMap["afterSetp5"]++
		if !node.IsNat1() {
			if node.StreamdPortHttp <= 0 || node.StreamdPortWt <= 0 || node.StreamdPortHttps <= 0 {
				nodeMap["portsChk"]++
				continue
			}
		}
		nodeMap["afterSetp6"]++
		if len(node.Schedules) != 0 {
			nodeMap["SchedulesChk"]++
			continue
		}
		nodeMap["afterSetp7"]++
		for _, ipInfo := range node.Ips {
			nodeMap["ipCnt"]++
			if ipInfo.Forbidden {
				nodeMap["ipForbidden"]++
				continue
			}
			nodeMap["afterSetp8"]++
			if ipInfo.OutMBps >= ipInfo.MaxOutMBps*0.93 {
				nodeMap["ipBwOverflow"]++
				continue
			}
			nodeMap["afterSetp9"]++
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				nodeMap["IsPrivateIP"]++
				continue
			}
			nodeMap["afterSetp10"]++
			if node.IsNat1() {
				nodeMap["nat1"]++
				continue
			}
			nodeMap["afterSetp11"]++
			if node.IsBanTransProv {
				nodeMap["IsBanTransProv"]++
				continue
			}
			nodeMap["afterSetp12"]++
			if ipInfo.IsIPv6 {
				nodeMap["IsIPv6"]++
				continue
			}
			nodeMap["afterSetp13"]++
		}
	}
	bytes, err := json.MarshalIndent(nodeMap, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

func LowBw2() {
	ignoreV6 := false
	ServingNodesIpCntStep1 := 0

	allNodes, err := public.GetAllRTNodes(logger, RedisCli)
	if err != nil {
		logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	rawDynamicNodes := make([]*public.RtNode, 0, len(allNodes))
	for _, node := range allNodes {
		if node.IsDynamic {
			rawDynamicNodes = append(rawDynamicNodes, node)
		}
	}

	dynamicNodes := rawDynamicNodes
	timeLimitCnt := 0
	nodeMap := make(map[string]int)
	for _, node := range dynamicNodes {
		if node == nil || !node.IsDynamic {
			continue
		}

		if !util.CheckNodeUsable(log.Logger, node, constTypeLive) {
			nodeMap["CheckNodeUsable"]++
			continue
		}
		nodeMap["step1"]++

		if !localUtil.CheckDynamicNodesPort(node) {
			nodeMap["checkDynamicNodesPort"]++
			continue
		}
		nodeMap["step2"]++

		if !localUtil.CheckCanScheduleOfTimeLimit(node, 3600) {
			nodeMap["checkCanScheduleOfTimeLimit"]++
			timeLimitCnt++
			continue
		}
		nodeMap["step3"]++

		for _, info := range node.Ips {
			if ignoreV6 && info.IsIPv6 {
				nodeMap["IsIPv6"]++
				continue
			}
			nodeMap["step4"]++

			if info.Forbidden {
				nodeMap["Forbidden"]++
				continue
			}
			/*
				if publicUtil.IsPrivateIP(info.Ip) {
					nodeMap["private"]++
					continue
				}
			*/

			if !publicUtil.IsPrivateIP(info.Ip) {
				if info.MaxOutMBps > 0 {
					if info.OutMBps > info.MaxOutMBps*0.93 {
						nodeMap["bwOverflow"]++
						continue
					}
				} else {
					nodeMap["bwOverflow"]++
					continue
				}
			}
			nodeMap["step5"]++

			ServingNodesIpCntStep1++
		}
	}
	logger.Info().Int("ServingNodesIpCntStep1", ServingNodesIpCntStep1).Any("nodeMap", nodeMap).Msg("lowbw2")
}

func DelDns() {
	cli, err := tencent_dnspod.NewTencentClient(conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	if conf.RecordId == "" {
		fmt.Println("need record id")
		return
	}
	if conf.Domain == "" {
		fmt.Println("need domain")
		return
	}
	recordIds := strings.Split(conf.RecordId, ",")
	intRecordIds := []uint64{}
	for _, recordId := range recordIds {
		u, err := strconv.ParseUint(recordId, 10, 64)
		if err != nil {
			fmt.Printf("转换错误: %v\n", err)
			return
		}
		intRecordIds = append(intRecordIds, u)
	}
	xl := xlog.NewDummyWithCtx(context.Background())

	for _, recordId := range intRecordIds {
		op := public.Operation{
			RecordId: recordId,
		}
		resp, err := cli.DeleteRaw(xl, &op, conf.Domain)
		if err != nil {
			fmt.Println(err)
			return
		}
		logger.Info().Any("resp", resp).Msg("del dns")
	}
}

func DnsLines() {
	cli, err := tencent_dnspod.NewTencentClient(conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := cli.GetLines(xl, conf.Domain, "")
	if err != nil {
		fmt.Println(err)
		return
	}
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return
	}
	fmt.Println(string(bytes))
}

func GenNodes() {
	redisCli := rediNewClusterClient(&rediClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}
	ipParser, err := ipdb.NewCity(conf.IPDB)
	if err != nil {
		fmt.Println(err)
		return
	}

	allNodes, err := public.GetAllRTNodes(logger, redisCli)
	if err != nil {
		logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	provIspNodesMap := make(map[string][]*public.RtNode)
	for _, node := range allNodes {
		if !node.IsDynamic {
			continue
		}
		if node.RuntimeStatus != "Serving" {
			continue
		}
		if node.Status != "normal" {
			continue
		}
		if node.IsNat1() {
			continue
		}
		if len(node.Schedules) != 0 {
			continue
		}
		if node.StreamdPortHttp <= 0 || node.StreamdPortWt <= 0 || node.StreamdPortHttps <= 0 {
			continue
		}
		isp, _, province := localUtil.GetNodeLocate(node, ipParser)
		if isp == "" || province == "" {
			continue
		}
		key := isp + "_" + province
		if _, ok := provIspNodesMap[key]; !ok {
			provIspNodesMap[key] = make([]*public.RtNode, 0)
		}
		if len(provIspNodesMap[key]) >= 2 {
			continue
		}
		provIspNodesMap[key] = append(provIspNodesMap[key], node)
	}

	delete(provIspNodesMap, "电信_北京")

	newMap := make(map[string][]*public.RtNode)
	idx := 0
	for key, nodes := range provIspNodesMap {
		if idx%3 == 0 {
			newMap[key] = nodes
		}
		idx++
	}

	idx = 0
	keys := []string{}
	for key := range newMap {
		if idx < 8 {
			keys = append(keys, key)
		}
		idx++
	}
	for _, key := range keys {
		delete(newMap, key)
	}

	total := 0
	for key, nodes := range newMap {
		fmt.Println(key)
		for _, node := range nodes {
			fmt.Println(node.Id)
			total++
		}
	}
	fmt.Println("total:", total)
	resultNodes := make([]*public.RtNode, 0, total)
	for _, nodes := range newMap {
		resultNodes = append(resultNodes, nodes...)
	}
	bytes, err := json.MarshalIndent(resultNodes, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	f := fmt.Sprintf("nodes_%d.json", time.Now().Unix())
	err = os.WriteFile(f, bytes, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func dumpNodes2() {
	redisCli := rediNewClusterClient(&rediClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}

	allNodes, err := public.GetAllRTNodes(logger, redisCli)
	if err != nil {
		logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	nodesMap := make(map[string]int)
	notServingIds := make([]string, 0)
	for _, node := range allNodes {
		if !node.IsDynamic {
			continue
		}
		nodesMap["动态节点个数"]++
		if node.IsBanTransProv {
			continue
		}
		hasv6 := false
		for _, ipInfo := range node.Ips {
			if ipInfo.IsIPv6 {
				hasv6 = true
				break
			}
		}
		if !hasv6 {
			continue
		}
		nodesMap["可出省动态节点个数"]++
		if node.RuntimeStatus != "Serving" {
			notServingIds = append(notServingIds, node.Id)
			continue
		}
		nodesMap["Serving可出省动态节点个数"]++
		ability, ok := node.Abilities["live"]
		if !ok || !ability.Can || ability.Frozen {
			continue
		}
		nodesMap["Abilities ok && Serving可出省动态节点个数"]++
		if _, ok = node.Services["live"]; !ok {
			continue
		}
		nodesMap["Services ok && Serving可出省动态节点个数"]++
	}
	bytes, err := json.MarshalIndent(nodesMap, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(bytes))
	fmt.Println(notServingIds)

}

func dumpNodeFromFile() {
	bytes, err := oReadFile("/Users/liyuanquan/workspace/tmp/nodes333.json")
	if err != nil {
		fmt.Println("read fail", "", err)
		return
	}
	nodes := []*public.RtNode{}
	if err := json.Unmarshal(bytes, &nodes); err != nil {
		logger.Err(err).Msg("dumpNodeFromFile")
		return
	}
	ipParser, err := ipdb.NewCity(conf.IPDB)
	if err != nil {
		logger.Err(err).Msg("dumpNodeFromFile")
		return
	}
	ipMap := make(map[string]map[string][]string)
	for _, node := range nodes {
		for _, ipInfo := range node.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			if node.IsBanTransProv {
				continue
			}
			isp, _, _ := localUtil.GetLocate(ipInfo.Ip, ipParser)
			if _, ok := ipMap[isp]; !ok {
				ipMap[isp] = make(map[string][]string)
			}
			ipVer := "v4"
			if ipInfo.IsIPv6 {
				ipVer = "v6"
			}
			if _, ok := ipMap[isp][ipVer]; !ok {
				ipMap[isp][ipVer] = make([]string, 0)
			}
			ipMap[isp][ipVer] = append(ipMap[isp][ipVer], ipInfo.Ip)
		}
	}
	for isp, m := range ipMap {
		fmt.Println(isp + ":")
		for ipVer, m1 := range m {
			fmt.Printf("\t%v:\n", ipVer)
			fmt.Printf("\t\t%d %+v\n", len(m1), m1)
		}
	}
}

func Retrans() {
	redisCli := rediNewClusterClient(&rediClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}

	allNodes, err := public.GetAllRTNodes(logger, redisCli)
	if err != nil {
		logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	machineMap := make(map[string]int)
	for _, node := range allNodes {
		if !node.IsDynamic {
			continue
		}
		if node.RuntimeStatus != "Serving" {
			continue
		}
		key := "tcpretran_result_" + node.Id
		samples, err := redisCli.ZRange(context.Background(), key, 0, -1).Result()
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, sample := range samples {
			rate, err := strconv.ParseFloat(sample, 64)
			if err != nil {
				logger.Error().Err(err).Msgf("GetNodeTcpRetranInfo parse rate failed, rate: %s", sample)
				continue
			}
			if node.StreamdPortHttp != 22222 {
				continue
			}
			if rate < 0.1 {
				continue
			}
			machineMap[node.MachineId]++
		}
	}
	fmt.Println(machineMap)
}

func Port80() {
	redisCli := rediNewClusterClient(&rediClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}
	allNodes, err := public.GetAllRTNodes(logger, redisCli)
	if err != nil {
		logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	nodes := []string{}
	for _, node := range allNodes {
		if node.NodeType != "80port" {
			continue
		}
		nodes = append(nodes, node.Id)
	}
	logger.Info().Int("len", len(nodes)).Any("nodes", nodes).Msg("80 port nodes")
}

type AlarmInfo struct {
	AlarmType int `xml:"AlarmType"`
}

type Alarm struct {
	GbId             string    `xml:"-" bson:"gbId" json:"gbId"`
	ChGbid           string    `xml:"-" bson:"chId" json:"chId"`
	AlarmPriority    int       `xml:"AlarmPriority" bson:"alarmPriority" json:"alarmPriority"`
	AlarmTimeStr     string    `xml:"AlarmTime" bson:"-" json:"-"` // only for xml parse
	AlarmTime        int       `xml:"-" bson:"alarmTime" json:"alarmTime"`
	AlarmMethod      int       `xml:"AlarmMethod" bson:"alarmMethod" json:"alarmMethod"`
	Info             AlarmInfo `xml:"Info,omitempty" bson:"-" json:"-"`
	DeviceState      string    `xml:"-" bson:"-" json:"deviceState"`
	AlarmType        int       `xml:"AlarmType" bson:"alarmType" json:"alarmType,omitempty"`
	ChName           string    `xml:"-" bson:"-" json:"chName"`
	ExpireAt         time.Time `xml:"-" bson:"expireAt"  json:"-" `
	AlarmDescription string    `xml:"AlarmDescription" bson:"-" json:"-"`
	Url              string    `xml:"Url,omitempty" bson:"-" json:"url,omitempty"`
}

type MobilePosition struct {
	MpGbId        string  `xml:"-" bson:"cbId" json:"gbid"`
	MpChGbId      string  `xml:"-" bson:"chId" json:"chid,omitempty"`
	Longitude     float64 `xml:"Longitude" bson:"longitude" json:"longitude"` // 经度
	Latitude      float64 `xml:"Latitude" bson:"latitude" json:"latitude"`    // 纬度
	Speed         float64 `xml:"Speed" bson:"speed" json:"speed,omitempty"`
	Direction     float64 `xml:"Direction" bson:"direction" json:"direction,omitempty"`
	Altitude      int     `xml:"Altitude" bson:"altitude" json:"altitude,omitempty"`
	CreateTime    int64   `xml:"-" bson:"createTime" json:"createTime"`
	CreateTimeStr string  `xml:"Time" bson:"-" json:"-"` // only for xml parse
}

type Item struct {
	ChId  string `xml:"DeviceID"`
	Start string `xml:"StartTime"`
	End   string `xml:"EndTime"`
	Type  string `xml:"Type"`
}

type RecordList struct {
	Num   string `xml:"Num,attr"`
	Items []Item `xml:"Item"`
}

type srsMessageXml struct {
	CmdType  string `xml:"CmdType"`
	SN       string `xml:"SN"`
	DeviceId string `xml:"DeviceID"`
	SumNum   int    `xml:"SumNum"`
	Alarm
	MobilePosition
	RecordList RecordList `xml:"RecordList,omitempty"`
	//PresetList PresetList `xml:"PresetList,omitempty"`
}

func Xml() {
	str := `<?xml version=\"1.0\"?>\r\n <Notify>\r\n <CmdType>Alarm</CmdType>\r\n <SN>1</SN>\r\n <DeviceID>34010000001310000001</DeviceID>\r\n <AlarmPriority>4</AlarmPriority>\r\n <AlarmTime>2025-01-23T15:58:16</AlarmTime>\r\n <AlarmMethod>5</AlarmMethod>\r\n <Info>\r\n <AlarmType>2</AlarmType>\r\n </Info>\r\n </Notify>`
	r := srsMessageXml{}
	decoder := xml.NewDecoder(byteNewReader([]byte(str)))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&r); err != nil {
		if stringIndex(err.Error(), "UTF-8") >= 0 {
			reader2 := transform.NewReader(byteNewReader([]byte(str)), simplifiedchinese.GBK.NewDecoder())
			d2, err := ioutil.ReadAll(reader2)
			if err != nil {
				fmt.Println("PostSrsClientsMessage err:", err)
				return
			}
			decoder := xml.NewDecoder(byteNewReader(d2))
			decoder.CharsetReader = charset.NewReaderLabel
			if err := decoder.Decode(&r); err != nil {
				fmt.Println("PostSrsClientsMessage err:", err)
				return
			}
		}
	}
	fmt.Printf("%+v\n", r)
}

func OnNode(node *public.RtNode) {
}

func OnIp(node *public.RtNode, ip *public.RtIpStatus) {
	if !node.IsNat1() {
		return
	}
	if node.StreamdPortHttp == 33333 {
		logger.Info().Str("nodeId", node.Id).Str("ip", ip.Ip).Int("oriPort", node.StreamdPortHttp).Msg("OnIp")
	}
	//logger.Info().Str("nodeId", node.Id).Str("ip", ip.Ip).Int("oriPort", node.StreamdPortHttp).Msg("OnIp")
	if ip.NatStreamdPortHttp != 33333 {
		return
	}
	logger.Info().Str("nodeId", node.Id).Str("ip", ip.Ip).Int("oriPort", node.StreamdPortHttp).Msg("OnIp")
}

func Nat1() {
	n := NewNodeTraverse(logger, s, conf.RedisAddrs)
	n.Traverse()
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func reverseRotationWithCenter(x1, y1, degrees, W, H float64) (x, y int) {
	theta := degreesToRadians(degrees)
	cosTheta := math.Cos(theta)
	sinTheta := math.Sin(theta)
	Cx := W / 2
	Cy := H / 2

	// 转换为相对中心点的坐标
	dx := x1 - Cx
	dy := y1 - Cy

	// 应用逆时针旋转（即原始顺时针旋转的逆操作）
	x_rotated := dx*cosTheta - dy*sinTheta
	y_rotated := dx*sinTheta + dy*cosTheta

	// 转换回原始坐标系
	x = int(x_rotated + Cx)
	y = int(y_rotated + Cy)
	return
}

var (
	cos30 = math.Sqrt(3) / 2 // cos(30°)
	sin30 = 0.5              // sin(30°)
)

// 计算旋转前的原始坐标
func GetOriginalCoord(x1, y1, w, h float64) (x, y float64) {
	// 计算中心点
	cx := w / 2
	cy := h / 2

	// 转换为相对中心点的坐标
	xRel := x1 - cx
	yRel := y1 - cy

	// 应用逆旋转矩阵 (逆时针30度)
	x = xRel*cos30 - yRel*sin30
	y = xRel*sin30 + yRel*cos30

	// 转换回原始坐标系
	x += cx
	y += cy

	return
}

func rotateBack(x1, y1, cx, cy float64, angleDeg float64) (float64, float64) {
	// 转换角度到弧度
	angleRad := angleDeg * math.Pi / 180

	// 步骤 1: 平移至以中心为原点
	xPrime := x1 - cx
	yPrime := y1 - cy

	// 步骤 2: 应用逆旋转变换（顺时针，所以使用负角度）
	xDoublePrime := xPrime*math.Cos(-angleRad) + yPrime*math.Sin(-angleRad)
	yDoublePrime := -xPrime*math.Sin(-angleRad) + yPrime*math.Cos(-angleRad)

	// 步骤 3: 平移回去
	x := xDoublePrime + cx
	y := yDoublePrime + cy

	return x, y
}

func rotateCoordinates(x1, y1, angle float64) (float64, float64) {
	// 将角度转换为弧度
	theta := angle * (math.Pi / 180)

	// 计算 cos 和 sin
	cosTheta := math.Cos(theta)
	sinTheta := math.Sin(theta)

	// 逆向旋转计算
	x := x1*cosTheta - y1*sinTheta
	y := x1*sinTheta + y1*cosTheta

	return x, y
}

// RotatePoint calculates the original coordinates (x, y) before rotation.
// x1, y1: Coordinates after rotation.
// angleDegrees: Rotation angle in degrees (clockwise).
// width, height: Width and height of the canva
func RotatePoint(x1, y1, angleDegrees, width, height float64) (x, y float64) {
	// Convert angle to radian
	angleRadians := angleDegrees * math.Pi / 180.0

	// Calculate the center of the canva
	centerX := width / 2.0
	centerY := height / 2.0

	// Translate the point to the origin (center of the canvas).
	x1Centered := x1 - centerX
	y1Centered := y1 - centerY

	// Perform reverse rotation.
	xCentered := x1Centered*math.Cos(angleRadians) + y1Centered*math.Sin(angleRadians)
	yCentered := -x1Centered*math.Sin(angleRadians) + y1Centered*math.Cos(angleRadians)

	// Translate back to the original coordinate system.
	x = xCentered + centerX
	y = yCentered + centerY

	return x, y
}

func Watermark() {
	rows := 5
	cols := 3
	xCoordinate := 104
	yCoordinate := 424
	xGap := 200
	yGap := 200
	textWidth := 240
	textHeight := 59
	canvasEdgeLength := 1468

	filters := fmt.Sprintf("color=c=%s:s=%dx%d[o1];", "black",
		canvasEdgeLength, canvasEdgeLength)
	cnt := 1
	//theta := 30 * math.Pi / 180
	//cosTheta := math.Cos(theta)
	//sinTheta := math.Sin(theta)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			xRot := xCoordinate + col*(xGap+textWidth)
			yRot := yCoordinate + row*(yGap+textHeight)

			//angle := Angle(float64(xRot), float64(yRot))

			//theta := angle * math.Pi / 180
			//cosTheta := math.Cos(theta)
			//sinTheta := math.Sin(theta)

			fmt.Println("addWatermarkArgs, row:", row, " col:", col, " xRot:", xRot, " yRot:", yRot)

			// 计算旋转前的坐标
			//xOrig := int(float64(xRot)*cosTheta + float64(yRot)*sinTheta)
			//yOrig := int(-float64(xRot)*sinTheta + float64(yRot)*cosTheta)
			//xOrig := xRot
			//yOrig := yRot
			//xOrig1, yOrig1 := rotateCoordinates(float64(xRot), float64(yRot), 30.0)

			//xOrig := int(xOrig1)
			//yOrig := int(yOrig1)

			x, y := RotatePoint(float64(xRot), float64(yRot), 30, 1468, 1468)
			xOrig := int(x)
			yOrig := int(y)

			// 坐标边界检查
			if xOrig < 0 {
				xOrig = 0
			} else if xOrig >= canvasEdgeLength {
				xOrig = canvasEdgeLength - 1
			}
			if yOrig < 0 {
				yOrig = 0
			} else if yOrig >= canvasEdgeLength {
				yOrig = canvasEdgeLength - 1
			}

			filters += fmt.Sprintf("[o%d]drawtext=text='%s%d':fontsize=%d:fontcolor=%s:x=%d:y=%d[o%d];",
				cnt, "文字水印测试", cnt, 30, "white", xOrig, yOrig, cnt+1)
			cnt++
		}
	}
	filters += fmt.Sprintf("[o%d]rotate=a=PI*30/180:fillcolor=%s@0[o%d];", cnt, "green", cnt+1)
	//cnt++
	//filters += fmt.Sprintf("[o%d]crop=%d:%d[o%d];", cnt, streamW, streamH, cnt+1)
	//filters += fmt.Sprintf("[o%d]crop=1280:720[o%d];", cnt, cnt+1)
	//cnt++
	//filters += fmt.Sprintf("[o%d]colorkey=%s:0.01:1[o%d];", cnt, watermark.CanvasBgColor, cnt+1)
	//cnt++
	//filters += fmt.Sprintf("[0:v][o%d]overlay=x=0:y=0[o%d]", cnt, cnt+1)
	cmd := exec.Command("ffmpeg", "-filter_complex", filters, "-map", "[o"+fmt.Sprintf("%d", cnt+1)+"]",
		"-frames:v", "1", "-update", "1", "/tmp/out.jpg")

	fmt.Println(cmd.String())

	var stdout, stderr byteBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()
	if err != nil {
		fmt.Println("执行命令出错:", err)
		fmt.Println(stderr.String()) // 打印错误输出
		return
	}

	// 打印命令的输出
	fmt.Println("标准输出:\n", stdout.String())
	/*
		txt := ""
		cnt := 1
		for i := 0; i < conf.Cnt; i++ {
			for j := 0; j < conf.Cnt+2; j++ {
				x := 60 + 450*i
				y := 50 + 450*j
				txt += fmt.Sprintf("[o%d]drawtext=text='文字水印测试%d':x=%d:y=%d:fontsize=20[o%d];\n", cnt, cnt, x, y, cnt+1)
				cnt++
			}
		}
		fmt.Println(txt)
	*/
}

func Angle(x, y float64) float64 {
	angleWithHorizontal := math.Atan2(y, x) * (180 / math.Pi)

	// 计算 d，即 m 与 n 连线与 f 之间的夹角
	d := angleWithHorizontal - 210

	// 确保角度在 0-360 度之间
	if d < 0 {
		d += 360
	}

	// 打印结果
	fmt.Printf("角度 d: %.2f 度\n", d)
	return d
}

func TestAngle() {
	Angle(100, 100)
	Angle(100, 200)
}

// 原代码中XML部分的双引号没有正确转义，导致字符串解析错误。
// 这里将XML部分的双引号转义，以确保字符串可以正确解析。
var msg = "MESSAGE sip:31011500991180004957 SIP/2.0\r\nVia: SIP/2.0/TCP 127.0.0.1:5061;rport\r\nFrom: <sip:31011500991180004957>\r\nTo: <sip:31011500991180004957>\r\nCall-ID: 31011500991180004957-1cptzpe-agk5dty5k53edhdaxkrr7olk4e\r\nCSeq: 1618 MESSAGE\r\nContent-Type: Application/MANSCDP+xml\r\nContent-Length: 272\r\n\r\n\r\n<?xml version=\"1.0\"?><Query><Token>363xsrn33ajh1</Token><GbServer>2</GbServer><CmdType>GetConfig</CmdType><Namespace>test_ikelink_com</Namespace><DeviceID>31011500991180004957</DeviceID><MessageId>31011500991180004957-1cptzpe-agk5dty5k53edhdaxkrr7olk4e</MessageId></Query>"

func SipRawReq() (int, string) {
	deflog.SetFlags(deflog.LstdFlags | deflog.Lshortfile)
	url := "http://10.70.67.39:7275/v1/namespaces/test_ikelink_com/devices/31011500991180004957/sipraw"
	headers := map[string]string{
		"authorization": "QiniuStub uid=1382871676",
		"Content-Type":  "application/json",
	}
	data := map[string]string{
		"msg": msg,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return 0, ""
	}

	req, err := http.NewRequest("POST", url, byteNewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return 0, ""
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return 0, ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return 0, ""
	}

	deflog.Println("Response status:", resp.Status)
	deflog.Println("Response body:", string(body))

	msg := fmt.Sprintf("Response status: %s, Response body: %s, headers: %+v", resp.Status, string(body), resp.Header)
	return resp.StatusCode, msg
}

func SipRaw() {
	// 每隔500ms调用一次SipRawReq，如果返回的code不是200的话，向企业微信发送告警
	tick := time.Tick(500 * time.Millisecond)
	for range tick {
		statusCode, msg := SipRawReq()
		if statusCode != 200 && statusCode != 612 {
			err := localUtil.SendWeChatAlert(msg)
			if err != nil {
				deflog.Println("Error sending WeChat alert:", err)
				return
			}
		}
	}
}

func SipRawPost() {
	url := "http://qvqiniuapi.com/v1/namespaces/test_ikelink_com/devices/31011500991180004957/sipraw"
	// 每隔500ms请求一次SipRawReq， 如果返回值不是599的话，向企业微信发送告警
	tick := time.Tick(500 * time.Millisecond)
	data := map[string]string{
		"msg": msg,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}
	for range tick {
		statusCode, msg, hdrs := localUtil.MikuHttpReqReturnHdr("POST", url, string(jsonData), Conf.Ak, conf.Sk)
		if statusCode != 200 && statusCode != 612 {
			content := fmt.Sprintf("Response status: %d, Response body: %s, headers: %+v", statusCode, msg, hdrs)
			err := localUtil.SendWeChatAlert(content)
			if err != nil {
				deflog.Println("Error sending WeChat alert:", err)
				return
			}
		}

	}
}

func SipRawGet() {
	url := "http://qvqiniuapi.com/v1/namespaces/test_ikelink_com/devices/31011500991180004957/sipraw"
	// 每隔500ms请求一次SipRawReq， 如果返回值不是599的话，向企业微信发送告警
	tick := time.Tick(500 * time.Millisecond)
	data := map[string]string{
		"msg": msg,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}
	for range tick {
		statusCode, msg, hdrs := localUtil.MikuHttpReqReturnHdr("POST", url, string(jsonData), conf.Ak, conf.Sk)
		if statusCode != 200 && statusCode != 612 {
			content := fmt.Sprintf("Response status: %d, Response body: %s, headers: %+v", statusCode, msg, hdrs)
			err := localUtil.SendWeChatAlert(content)
			if err != nil {
				deflog.Println("Error sending WeChat alert:", err)
				return
			}
		}

	}
}

// ClientInfo 定义客户端信息结构体
type ClientInfo struct {
	OSName         string `json:"osName"`
	OSVersion      string `json:"osVersion"`
	BrowserName    string `json:"browserName"`
	BrowserVersion string `json:"browserVersion"`
	SDKVersion     string `json:"SDK_VERSION"`
}

// EditPrompt 定义编辑提示的结构体
type EditPrompt struct {
	StreamURL string     `json:"streamurl"`
	ClientIP  ClientInfo `json:"clientip"`
	SDP       string     `json:"sdp"`
}

var sdp = "v=0\r\no=- 1487257410276300399 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\na=group:BUNDLE 0 1\r\na=extmap-allow-mixed\r\na=msid-semantic: WMS\r\nm=audio 9 UDP/TLS/RTP/SAVPF 111 63 9 0 8 13 110 126\r\nc=IN IP4 0.0.0.0\r\na=rtcp:9 IN IP4 0.0.0.0\r\na=ice-ufrag:ebKC\r\na=ice-pwd:4UmtO3aLN2jYrIJU9qhIOk1K\r\na=ice-options:trickle\r\na=fingerprint:sha-256 39:AB:A8:B2:9C:C9:A9:0A:8C:E7:20:BA:12:1F:4C:0C:CB:A8:71:5C:3D:27:AD:C3:97:A8:2B:FA:AD:54:1C:A1\r\na=setup:actpass\r\na=mid:0\r\na=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level\r\na=extmap:2 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time\r\na=extmap:3 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01\r\na=extmap:4 urn:ietf:params:rtp-hdrext:sdes:mid\r\na=recvonly\r\na=rtcp-mux\r\na=rtcp-rsize\r\na=rtpmap:111 opus/48000/2\r\na=rtcp-fb:111 transport-cc\r\na=fmtp:111 minptime=10;useinbandfec=1\r\na=rtpmap:63 red/48000/2\r\na=fmtp:63 111/111\r\na=rtpmap:9 G722/8000\r\na=rtpmap:0 PCMU/8000\r\na=rtpmap:8 PCMA/8000\r\na=rtpmap:13 CN/8000\r\na=rtpmap:110 telephone-event/48000\r\na=rtpmap:126 telephone-event/8000\r\nm=video 9 UDP/TLS/RTP/SAVPF 96 97 98 99 100 101 35 36 37 38 103 104 107 108 109 114 115 116 117 118 39 40 41 42 43 44 45 46 47 48 119 120 121 122 49 50 51 52 123 124 125 53\r\nc=IN IP4 0.0.0.0\r\na=rtcp:9 IN IP4 0.0.0.0\r\na=ice-ufrag:ebKC\r\na=ice-pwd:4UmtO3aLN2jYrIJU9qhIOk1K\r\na=ice-options:trickle\r\na=fingerprint:sha-256 39:AB:A8:B2:9C:C9:A9:0A:8C:E7:20:BA:12:1F:4C:0C:CB:A8:71:5C:3D:27:AD:C3:97:A8:2B:FA:AD:54:1C:A1\r\na=setup:actpass\r\na=mid:1\r\na=extmap:14 urn:ietf:params:rtp-hdrext:toffset\r\na=extmap:2 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time\r\na=extmap:13 urn:3gpp:video-orientation\r\na=extmap:3 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01\r\na=extmap:5 http://www.webrtc.org/experiments/rtp-hdrext/playout-delay\r\na=extmap:6 http://www.webrtc.org/experiments/rtp-hdrext/video-content-type\r\na=extmap:7 http://www.webrtc.org/experiments/rtp-hdrext/video-timing\r\na=extmap:8 http://www.webrtc.org/experiments/rtp-hdrext/color-space\r\na=extmap:4 urn:ietf:params:rtp-hdrext:sdes:mid\r\na=extmap:10 urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id\r\na=extmap:11 urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id\r\na=recvonly\r\na=rtcp-mux\r\na=rtcp-rsize\r\na=rtpmap:96 VP8/90000\r\na=rtcp-fb:96 goog-remb\r\na=rtcp-fb:96 transport-cc\r\na=rtcp-fb:96 ccm fir\r\na=rtcp-fb:96 nack\r\na=rtcp-fb:96 nack pli\r\na=rtpmap:97 rtx/90000\r\na=fmtp:97 apt=96\r\na=rtpmap:98 VP9/90000\r\na=rtcp-fb:98 goog-remb\r\na=rtcp-fb:98 transport-cc\r\na=rtcp-fb:98 ccm fir\r\na=rtcp-fb:98 nack\r\na=rtcp-fb:98 nack pli\r\na=fmtp:98 profile-id=0\r\na=rtpmap:99 rtx/90000\r\na=fmtp:99 apt=98\r\na=rtpmap:100 VP9/90000\r\na=rtcp-fb:100 goog-remb\r\na=rtcp-fb:100 transport-cc\r\na=rtcp-fb:100 ccm fir\r\na=rtcp-fb:100 nack\r\na=rtcp-fb:100 nack pli\r\na=fmtp:100 profile-id=2\r\na=rtpmap:101 rtx/90000\r\na=fmtp:101 apt=100\r\na=rtpmap:35 VP9/90000\r\na=rtcp-fb:35 goog-remb\r\na=rtcp-fb:35 transport-cc\r\na=rtcp-fb:35 ccm fir\r\na=rtcp-fb:35 nack\r\na=rtcp-fb:35 nack pli\r\na=fmtp:35 profile-id=1\r\na=rtpmap:36 rtx/90000\r\na=fmtp:36 apt=35\r\na=rtpmap:37 VP9/90000\r\na=rtcp-fb:37 goog-remb\r\na=rtcp-fb:37 transport-cc\r\na=rtcp-fb:37 ccm fir\r\na=rtcp-fb:37 nack\r\na=rtcp-fb:37 nack pli\r\na=fmtp:37 profile-id=3\r\na=rtpmap:38 rtx/90000\r\na=fmtp:38 apt=37\r\na=rtpmap:103 H264/90000\r\na=rtcp-fb:103 goog-remb\r\na=rtcp-fb:103 transport-cc\r\na=rtcp-fb:103 ccm fir\r\na=rtcp-fb:103 nack\r\na=rtcp-fb:103 nack pli\r\na=fmtp:103 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f\r\na=rtpmap:104 rtx/90000\r\na=fmtp:104 apt=103\r\na=rtpmap:107 H264/90000\r\na=rtcp-fb:107 goog-remb\r\na=rtcp-fb:107 transport-cc\r\na=rtcp-fb:107 ccm fir\r\na=rtcp-fb:107 nack\r\na=rtcp-fb:107 nack pli\r\na=fmtp:107 level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f\r\na=rtpmap:108 rtx/90000\r\na=fmtp:108 apt=107\r\na=rtpmap:109 H264/90000\r\na=rtcp-fb:109 goog-remb\r\na=rtcp-fb:109 transport-cc\r\na=rtcp-fb:109 ccm fir\r\na=rtcp-fb:109 nack\r\na=rtcp-fb:109 nack pli\r\na=fmtp:109 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f\r\na=rtpmap:114 rtx/90000\r\na=fmtp:114 apt=109\r\na=rtpmap:115 H264/90000\r\na=rtcp-fb:115 goog-remb\r\na=rtcp-fb:115 transport-cc\r\na=rtcp-fb:115 ccm fir\r\na=rtcp-fb:115 nack\r\na=rtcp-fb:115 nack pli\r\na=fmtp:115 level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f\r\na=rtpmap:116 rtx/90000\r\na=fmtp:116 apt=115\r\na=rtpmap:117 H264/90000\r\na=rtcp-fb:117 goog-remb\r\na=rtcp-fb:117 transport-cc\r\na=rtcp-fb:117 ccm fir\r\na=rtcp-fb:117 nack\r\na=rtcp-fb:117 nack pli\r\na=fmtp:117 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=4d001f\r\na=rtpmap:118 rtx/90000\r\na=fmtp:118 apt=117\r\na=rtpmap:39 H264/90000\r\na=rtcp-fb:39 goog-remb\r\na=rtcp-fb:39 transport-cc\r\na=rtcp-fb:39 ccm fir\r\na=rtcp-fb:39 nack\r\na=rtcp-fb:39 nack pli\r\na=fmtp:39 level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=4d001f\r\na=rtpmap:40 rtx/90000\r\na=fmtp:40 apt=39\r\na=rtpmap:41 H264/90000\r\na=rtcp-fb:41 goog-remb\r\na=rtcp-fb:41 transport-cc\r\na=rtcp-fb:41 ccm fir\r\na=rtcp-fb:41 nack\r\na=rtcp-fb:41 nack pli\r\na=fmtp:41 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=f4001f\r\na=rtpmap:42 rtx/90000\r\na=fmtp:42 apt=41\r\na=rtpmap:43 H264/90000\r\na=rtcp-fb:43 goog-remb\r\na=rtcp-fb:43 transport-cc\r\na=rtcp-fb:43 ccm fir\r\na=rtcp-fb:43 nack\r\na=rtcp-fb:43 nack pli\r\na=fmtp:43 level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=f4001f\r\na=rtpmap:44 rtx/90000\r\na=fmtp:44 apt=43\r\na=rtpmap:45 AV1/90000\r\na=rtcp-fb:45 goog-remb\r\na=rtcp-fb:45 transport-cc\r\na=rtcp-fb:45 ccm fir\r\na=rtcp-fb:45 nack\r\na=rtcp-fb:45 nack pli\r\na=fmtp:45 level-idx=5;profile=0;tier=0\r\na=rtpmap:46 rtx/90000\r\na=fmtp:46 apt=45\r\na=rtpmap:47 AV1/90000\r\na=rtcp-fb:47 goog-remb\r\na=rtcp-fb:47 transport-cc\r\na=rtcp-fb:47 ccm fir\r\na=rtcp-fb:47 nack\r\na=rtcp-fb:47 nack pli\r\na=fmtp:47 level-idx=5;profile=1;tier=0\r\na=rtpmap:48 rtx/90000\r\na=fmtp:48 apt=47\r\na=rtpmap:119 H264/90000\r\na=rtcp-fb:119 goog-remb\r\na=rtcp-fb:119 transport-cc\r\na=rtcp-fb:119 ccm fir\r\na=rtcp-fb:119 nack\r\na=rtcp-fb:119 nack pli\r\na=fmtp:119 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=64001f\r\na=rtpmap:120 rtx/90000\r\na=fmtp:120 apt=119\r\na=rtpmap:121 H264/90000\r\na=rtcp-fb:121 goog-remb\r\na=rtcp-fb:121 transport-cc\r\na=rtcp-fb:121 ccm fir\r\na=rtcp-fb:121 nack\r\na=rtcp-fb:121 nack pli\r\na=fmtp:121 level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=64001f\r\na=rtpmap:122 rtx/90000\r\na=fmtp:122 apt=121\r\na=rtpmap:49 H265/90000\r\na=rtcp-fb:49 goog-remb\r\na=rtcp-fb:49 transport-cc\r\na=rtcp-fb:49 ccm fir\r\na=rtcp-fb:49 nack\r\na=rtcp-fb:49 nack pli\r\na=fmtp:49 level-id=180;profile-id=1;tier-flag=0;tx-mode=SRST\r\na=rtpmap:50 rtx/90000\r\na=fmtp:50 apt=49\r\na=rtpmap:51 H265/90000\r\na=rtcp-fb:51 goog-remb\r\na=rtcp-fb:51 transport-cc\r\na=rtcp-fb:51 ccm fir\r\na=rtcp-fb:51 nack\r\na=rtcp-fb:51 nack pli\r\na=fmtp:51 level-id=180;profile-id=2;tier-flag=0;tx-mode=SRST\r\na=rtpmap:52 rtx/90000\r\na=fmtp:52 apt=51\r\na=rtpmap:123 red/90000\r\na=rtpmap:124 rtx/90000\r\na=fmtp:124 apt=123\r\na=rtpmap:125 ulpfec/90000\r\na=rtpmap:53 flexfec-03/90000\r\na=rtcp-fb:53 goog-remb\r\na=rtcp-fb:53 transport-cc\r\na=fmtp:53 repair-window=10000000\r\n"

func testRtc() {
	clientIp := ClientInfo{
		OSName:         "Mac OS",
		OSVersion:      "10.15.7",
		BrowserName:    "Chrome",
		BrowserVersion: "136.0.0.0",
		SDKVersion:     "1.1.1-alpha.2",
	}

	for i := 0; i < 100000; i++ {
		addr := fmt.Sprintf("webrtc://127.0.0.1:2985/live/teststream%d", i)
		editPrompt := EditPrompt{
			StreamURL: addr,
			ClientIP:  clientIp,
			SDP:       sdp,
		}
		jsonData, err := json.Marshal(editPrompt)
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			return
		}
		url := "http://127.0.0.1:2985/rtc/v1/play"
		// 创建一个 HTTP POST 请求
		req, err := http.NewRequest("POST", url, byteNewBuffer(jsonData))
		if err != nil {
			fmt.Printf("创建请求出错: %v\n", err)
			return
		}

		// 设置请求头，指定 Content-Type 为 application/json
		req.Header.Set("Content-Type", "application/json")

		// 创建 HTTP 客户端并发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("发送请求出错: %v\n", err)
			return
		}
		defer resp.Body.Close()

		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("读取响应体出错: %v\n", err)
			return
		}

		// 打印响应状态码和响应体
		fmt.Printf("响应状态码: %d\n", resp.StatusCode)
		fmt.Printf("响应体: %s\n", string(body))
	}
}

func rtpTest() {
	ssrc := createVideoChannel("127.0.0.1", 2985)
	if ssrc == 0 {
		fmt.Println("createVideoChannel failed")
		return
	}
	fmt.Println("createVideoChannel success", ssrc)
	sendVideoRtp(ssrc)
	time.Sleep(1 * time.Second)
	ssrc = createVideoChannel("127.0.0.1", 2985)
	if ssrc == 0 {
		fmt.Println("createVideoChannel failed")
		return
	}
	fmt.Println("createVideoChannel 2 success", ssrc)
	sendVideoRtp(ssrc)
}

func ReportChk() {
	RedisCli = rediNewClusterClient(&rediClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := RedisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println("Ping redis failed", err)
		return
	}
	nodetraverse.RegisterReportChk()
	//nodetraverse.Traverse(conf.RedisAddrs, conf.IPDB, s)
}

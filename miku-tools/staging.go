package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	schedUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/cmd/sched/model"
	"github.com/qbox/mikud-live/common"
	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/exp/rand"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 临时的代码放到这里

func (s *Parser) Staging() {
	switch s.conf.SubCmd {
	case "getpcdn":
		s.getPcdnFromSchedAPI(true, false)
	case "volc":
		s.fetchVolcOriginUrl()
	case "ipv6":
		s.dumpNodeIpv4v6Dis()
	case "hashRingMap":
		s.HashRingMap()
	case "refactor":
		s.Refactor()
	case "load":
		s.Load()
	case "exec":
		s.Exec()
	case "dns":
		s.Dnspod()
	case "log":
		m := make(map[string]string)
		m["aaa"] = "hello"
		m["bbb"] = "world"
		m["ccc"] = "foo"
		s.logger.Info().Any("mm", m).Msg("test")
	case "lowbw":
		//s.LowBw()
		//s.buildAllNodesMap()
		s.DumpNodes()
	case "dnsrecords":
		s.DnsRecords()
	case "lowbw2":
		s.buildAllNodesMap()
		s.LowBw2()
	case "deldns":
		s.DelDns()
	case "lines":
		s.DnsLines()
	case "nodes":
		s.GenNodes()
	case "dump":
		s.dumpNodes2()
	case "dumpNodeFromFile":
		s.dumpNodeFromFile()
	case "retrans":
		s.Retrans()
	case "80port":
		s.Port80()
	case "xml":
		s.Xml()
	}
}

type NodeDis struct {
	Ipv4Cnt int
	Ipv6Cnt int
}

func (s *Parser) dumpNodeIpv4v6Dis() {
	areaIpv4v6CntMap := make(map[string]*NodeDis)
	for _, node := range s.allNodesMap {
		for _, ip := range node.Ips {
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			if !IsPublicIPAddress(ip.Ip) {
				continue
			}
			isp, area, _ := getLocate(ip.Ip, s.IpParser)
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
		areaIpv6PercentMap[areaIsp] = nodeDis.Ipv6Cnt * 100 / (nodeDis.Ipv4Cnt + nodeDis.Ipv6Cnt)
	}
	pairs := SortIntMap(areaIpv6PercentMap)
	DumpSlice(pairs)

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
	case consts.BwSortByRemainRatio:
		weightList := make([]int, 0)
		for _, pair := range h.rawPairList {
			ratio2Weight := int(pair.RemainRatio * 100)
			h.totalWeight += ratio2Weight
			weightList = append(weightList, ratio2Weight)
		}
		h.weightList = weightList
		h.preSumWeight = calPreSum(weightList)
	//case consts.BwSortByRemainBw:
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
		//go alarm.SendWeChatAlarm(consts.WechatAlarmUrl, "HashRing", "", msg, consts.ErrInvalidTotalWeight)

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
	h.nodes.Sort()
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
	case consts.AlgorithmConsistentHash:
		return h.GetNodeByConsistentHash(streamId)
	case consts.AlgorithmBwShare:
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

		if sortType == consts.BwSortByRemainRatio {
			sort.Sort(model.IpListByRemainRatio(pairList[i].Ips))
		} else {
			sort.Sort(model.IpListByRemainBw(pairList[i].Ips))
		}
	}

	if sortType == consts.BwSortByRemainRatio {
		sort.Sort(model.PairListByRemainRatio(pairList))
	} else {
		sort.Sort(model.PairListByRemainBw(pairList))
	}
}

func (s *Parser) HashRingMap() {
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

func (s *Parser) Refactor() {
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

func (s *Parser) Load() {
	cnt := 0
	total := 0
	nodes := make(map[string]string)
	for _, node := range s.allNodesMap {
		if node.RuntimeStatus != "Serving" {
			continue
		}
		if !util.CheckNodeUsable(zlog.Logger, node, consts.TypeLive) {
			continue
		}

		if node.StreamdPorts.Http <= 0 || node.StreamdPorts.Https <= 0 || node.StreamdPorts.Wt <= 0 {
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
	s.logger.Info().Int("cnt", cnt).Int("total", total).Msg("")
	for nodeid, ip := range nodes {
		s.logger.Info().Str("node", nodeid).Str("ip", ip).Msg("")
	}
}

func (s *Parser) Exec() {
	cmd := exec.Command("jumpboxCmdNew", "redis-cli -h 10.70.60.31 -p 8200 -c --raw hgetall mik_netprobe_runtime_nodes_map")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println("the output is:", string(output))
}

func (s *Parser) Dnspod() {
	cli, err := tencent_dnspod.NewTencentClient(s.conf.DnsPod)
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
	//resp, err := cli.GetLines(xl, "mikudns.com", "")
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
		resp1, err := cli.CreateRaw(xl, &op, "qnrd.volclivedvs.com", "abc123hello")
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
		resp1, err = cli.CreateRaw(xl, &op1, "qnrd.volclivedvs.com", "abc123hello")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(resp1)
	*/

}

func (s *Parser) getAllNodes() string {
	raw := "redis-cli -h 10.70.60.31 -p 8200 -c --raw hgetall mik_netprobe_runtime_nodes_map"
	cmd := exec.Command("jumpboxCmdNew", raw)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return ""
	}
	//fmt.Println("the output is:", string(output))
	pos := strings.Index(string(output), raw)
	if pos == -1 {
		fmt.Println("find mik_netprobe_runtime_nodes_map && exti err")
		return ""
	}
	pos += len(raw) + 2
	str := string(output)[pos:]

	pos = strings.Index(str, raw)
	if pos == -1 {
		fmt.Println("find mik_netprobe_runtime_nodes_map && exti err")
		return ""
	}
	pos += len(raw) + 10
	return string(str)[pos:]
}

func (s *Parser) LowBw() {
	file := "/tmp/nodes.json"
	_, err := os.Stat(file)

	output := ""
	if os.IsNotExist(err) || s.conf.Force {
		output = s.getAllNodes()
		err := ioutil.WriteFile(file, []byte(output), 0644)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else if err != nil {
		fmt.Printf("检查文件 %s 时发生错误: %s\n", file, err)
		return
	} else {
		s.logger.Info().Str("file", file).Msg("read from file")
		bytes, err := ioutil.ReadFile(file)
		if err != nil {
			s.logger.Error().Err(err).Str("file", file).Msg("read file err")
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
			s.logger.Error().Err(err).Int("i", i).Msg("unmarshal")
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
			s.logger.Warn().Msgf("[CheckNodeUsable] not pass, nodeId:%s, machineId:%s, isDynamic:%t, abilities:%+v",
				node.Id, node.MachineId, node.IsDynamic, node.Abilities)
			continue
		}

		if _, ok = node.Services["live"]; !ok {
			s.logger.Warn().Msgf("[CheckNodeUsable] not pass, nodeId:%s, machineId:%s, isDynamic:%t, services:%+v",
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
	s.logger.Info().
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

func (s *Parser) isProvinceLine(line string) bool {
	for _, province := range Provinces {
		if strings.Contains(line, province) {
			return true
		}
	}
	return false
}

func (s *Parser) DnsRecords() {
	cli, err := tencent_dnspod.NewTencentClient(s.conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := cli.GetRecords(xl, s.conf.Domain, "", "", 0)
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
			err = os.WriteFile("/tmp/out.log", bytes, 0644)
			if err != nil {
				fmt.Println(err)
				return
			}
	*/
	lineMap := make(map[string]map[string][]string)

	totalV4Cnt := 0
	totalV6Cnt := 0

	areaMap := make(map[string]map[string][]string)
	for _, record := range resp.RecordList {
		if record.Name == nil {
			continue
		}
		if *record.Name != s.conf.Name {
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
		lineMap[*record.Line][*record.Type] = append(lineMap[*record.Line][*record.Type], *record.Value)

		if s.isProvinceLine(*record.Line) {
			if *record.Type == "A" {
				totalV4Cnt++
			}
			if *record.Type == "AAAA" {
				totalV6Cnt++
			}
		}

		isp := ""
		prov := ""
		for _, isp = range Isps {
			if strings.Contains(*record.Line, isp) {
				prov = strings.ReplaceAll(*record.Line, isp, "")
				break
			}
		}
		if isp == "" || prov == "" {
			s.logger.Error().Str("line", *record.Line).Msg("DnsRecords")
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
	}
	for line, typeMap := range lineMap {
		fmt.Printf("%s:\n", line)
		for t, ips := range typeMap {
			fmt.Printf("\t%s:\n", t)
			fmt.Printf("\t\t%d %+v\n", len(ips), ips)
		}
	}
	fmt.Println("totalV4Cnt:", totalV4Cnt)
	fmt.Println("totalV6Cnt:", totalV6Cnt)

	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      s.conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err = redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}

	allNodes, err := public.GetAllRTNodes(s.logger, redisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	for areaIsp, typeMap := range areaMap {
		fmt.Printf("%s:\n", areaIsp)
		for t, ips := range typeMap {
			fmt.Printf("\t%s:\n", t)
			fmt.Printf("\t\t%d %+v\n", len(ips), ips)
			bw := s.calcBw(allNodes, ips)
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
}

func (s *Parser) calcBw(allNodes []*public.RtNode, ips []string) (totalBw float64) {
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

func (s *Parser) DumpNodes() {
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      s.conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		zlog.Fatal().Err(err).Msg("")
	}

	allNodes, err := public.GetAllRTNodes(s.logger, redisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
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
			if node.StreamdPorts.Http <= 0 || node.StreamdPorts.Wt <= 0 || node.StreamdPorts.Https <= 0 {
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

func (s *Parser) LowBw2() {
	ignoreV6 := false
	ServingNodesIpCntStep1 := 0

	allNodes, err := public.GetAllRTNodes(s.logger, s.RedisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
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

		if !util.CheckNodeUsable(log.Logger, node, consts.TypeLive) {
			nodeMap["CheckNodeUsable"]++
			continue
		}
		nodeMap["step1"]++

		if !checkDynamicNodesPort(node) {
			nodeMap["checkDynamicNodesPort"]++
			continue
		}
		nodeMap["step2"]++

		if !checkCanScheduleOfTimeLimit(node, 3600) {
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
	s.logger.Info().Int("ServingNodesIpCntStep1", ServingNodesIpCntStep1).Any("nodeMap", nodeMap).Msg("lowbw2")
}

func (s *Parser) DelDns() {
	cli, err := tencent_dnspod.NewTencentClient(s.conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	if s.conf.RecordId == "" {
		fmt.Println("need record id")
		return
	}
	if s.conf.Domain == "" {
		fmt.Println("need domain")
		return
	}
	recordIds := strings.Split(s.conf.RecordId, ",")
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
		resp, err := cli.DeleteRaw(xl, &op, s.conf.Domain)
		if err != nil {
			fmt.Println(err)
			return
		}
		s.logger.Info().Any("resp", resp).Msg("del dns")
	}
}

func (s *Parser) DnsLines() {
	cli, err := tencent_dnspod.NewTencentClient(s.conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := cli.GetLines(xl, s.conf.Domain, "")
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

func (s *Parser) GenNodes() {
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      s.conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}
	ipParser, err := ipdb.NewCity(s.conf.IPDB)
	if err != nil {
		fmt.Println(err)
		return
	}

	allNodes, err := public.GetAllRTNodes(s.logger, redisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
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
		if node.StreamdPorts.Http <= 0 || node.StreamdPorts.Wt <= 0 || node.StreamdPorts.Https <= 0 {
			continue
		}
		isp, _, province := getNodeLocate(node, ipParser)
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

func (s *Parser) dumpNodes2() {
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      s.conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}

	allNodes, err := public.GetAllRTNodes(s.logger, redisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
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

func (s *Parser) dumpNodeFromFile() {
	bytes, err := os.ReadFile("/Users/liyuanquan/workspace/tmp/nodes333.json")
	if err != nil {
		fmt.Println("read fail", "", err)
		return
	}
	nodes := []*public.RtNode{}
	if err := json.Unmarshal(bytes, &nodes); err != nil {
		s.logger.Err(err).Msg("dumpNodeFromFile")
		return
	}
	ipParser, err := ipdb.NewCity(s.conf.IPDB)
	if err != nil {
		s.logger.Err(err).Msg("dumpNodeFromFile")
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
			isp, _, _ := getLocate(ipInfo.Ip, ipParser)
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

func (s *Parser) Retrans() {
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      s.conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}

	allNodes, err := public.GetAllRTNodes(s.logger, redisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
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
				s.logger.Error().Err(err).Msgf("GetNodeTcpRetranInfo parse rate failed, rate: %s", sample)
				continue
			}
			if node.StreamdPorts.Http != 22222 {
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

func (s *Parser) Port80() {
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      s.conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println(err)
		return
	}
	allNodes, err := public.GetAllRTNodes(s.logger, redisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	nodes := []string{}
	for _, node := range allNodes {
		if node.NodeType != "80port" {
			continue
		}
		nodes = append(nodes, node.Id)
	}
	s.logger.Info().Int("len", len(nodes)).Any("nodes", nodes).Msg("80 port nodes")
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

func (s *Parser) Xml() {
	str := `<?xml version=\"1.0\"?>\r\n <Notify>\r\n <CmdType>Alarm</CmdType>\r\n <SN>1</SN>\r\n <DeviceID>34010000001310000001</DeviceID>\r\n <AlarmPriority>4</AlarmPriority>\r\n <AlarmTime>2025-01-23T15:58:16</AlarmTime>\r\n <AlarmMethod>5</AlarmMethod>\r\n <Info>\r\n <AlarmType>2</AlarmType>\r\n </Info>\r\n </Notify>`
	r := srsMessageXml{}
	decoder := xml.NewDecoder(bytes.NewReader([]byte(str)))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&r); err != nil {
		if strings.Index(err.Error(), "UTF-8") >= 0 {
			reader2 := transform.NewReader(bytes.NewReader([]byte(str)), simplifiedchinese.GBK.NewDecoder())
			d2, err := ioutil.ReadAll(reader2)
			if err != nil {
				fmt.Println("PostSrsClientsMessage err:", err)
				return
			}
			decoder := xml.NewDecoder(bytes.NewReader(d2))
			decoder.CharsetReader = charset.NewReaderLabel
			if err := decoder.Decode(&r); err != nil {
				fmt.Println("PostSrsClientsMessage err:", err)
				return
			}
		}
	}
	fmt.Printf("%+v\n", r)
}

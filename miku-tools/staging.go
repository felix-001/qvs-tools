package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/cmd/sched/model"
	"github.com/qbox/mikud-live/common"
	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
	"github.com/rs/zerolog/log"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/exp/rand"
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
	/*
		//fmt.Println(cli)
		xl := xlog.NewDummyWithCtx(context.Background())
		//resp, err := cli.GetLines(xl, "zeicaefiegoh.com", "")
		resp, err := cli.GetLines(xl, "qnrd.volclivedvs.com", "")
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
		resp1, err := cli.GetDomain(xl, "ietheivaicai.com")
		if err != nil {
			fmt.Println(err)
			return
		}

		bytes, err := json.MarshalIndent(resp1, "", "  ")
		if err != nil {
			return
		}
		fmt.Println(string(bytes))
	*/

	/*
		resp, err := cli.GetRecords(xl, "zeicaefiegoh.com", "", "", 0)
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

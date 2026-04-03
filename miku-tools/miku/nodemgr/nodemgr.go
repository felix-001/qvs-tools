package nodemgr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
	"os"
	"strings"
	"time"

	commonModel "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/rs/zerolog"
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

func (m *NodeMgr) LoadNodes() {
	log.Println("LoadNodes")
	if m.LoadNodesFromFile("/tmp/allnodes.json") {
		return
	}
	log.Println("redis:", m.resources.Redis)
	allNodes, err := commonModel.GetAllRTNodes(zerolog.Logger{}, m.resources.Redis)
	if err != nil {
		fmt.Println("LoadNodes GetAllRTNodes err:", err)
		return
	}
	allNodesMap := make(map[string]*commonModel.RtNode)
	for _, node := range allNodes {
		allNodesMap[node.Id] = node
	}
	m.allNodesMap = allNodesMap
	m.filterMgr.SetAllNodes(allNodes)
	log.Println("LoadNodes 成功")
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

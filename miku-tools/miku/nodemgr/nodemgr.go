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
		fmt.Println("从/tmp/allnodes.json文件加载节点信息成功")
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

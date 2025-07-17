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
	return &NodeMgr{
		allRootNodesMapByNodeId: make(map[string]*commonModel.RtNode),
		allNodesMap:             make(map[string]*commonModel.RtNode),
		filterMgr:               NewFilterMgr(),
	}
}

func (m *NodeMgr) SetConf(conf *config.Config) {
	m.conf = conf
}

func (m *NodeMgr) SetResources(resources resources.Resources) {
	m.resources = resources
}

func (m *NodeMgr) GetRootNodeByNodeId(nodeId string) *commonModel.RtNode {
	return m.allRootNodesMapByNodeId[nodeId]
}

func (m *NodeMgr) GetNodeByNodeId(nodeId string) *commonModel.RtNode {
	return m.allNodesMap[nodeId]
}

func (m *NodeMgr) GetNodeByIp() {
	for _, node := range m.allNodesMap {
		for _, ipInfo := range node.Ips {
			if ipInfo.Ip == m.conf.Ip {
				_, ok := m.allRootNodesMapByNodeId[node.Id]
				fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "isRoot:", ok)
				break
			}
		}
	}
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
	isp, _, _ := util.GetLocate(ip.Ip, b.ipParser)
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
	m.Traverse()
	log.Println("avialiableNodeCnt:", b.avialiableNodeCnt,
		"avialiableIpCnt", b.avialiableIpCnt)
	log.Println("ispAvialiableBwMap len:", len(b.ispAvialiableBwMap))
	for k, bw := range b.ispAvialiableBwMap {
		log.Printf("isp: %s, bw: %.1fGbps", k, bw)
	}
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
		fmt.Println("从/tmp/allNodes.json文件加载节点信息成功")
		return true
	}
	return false
}

func (m *NodeMgr) LoadNodes() {
	fmt.Println("LoadNodes")
	if m.LoadNodesFromFile("/tmp/allNodes.json") {
		return
	}
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
}

func (m *NodeMgr) DumpNodes() {
	data, err := json.Marshal(m.allNodesMap)
	if err != nil {
		fmt.Println("DumpNodes Marshal err:", err)
		return
	}
	if err := os.WriteFile("allnodes.json", data, 0644); err != nil {
		fmt.Println("DumpNodes WriteFile err:", err)
		return
	}
	util.UploadFile("allnodes.json")
	fmt.Println("DumpNodes 成功")
}

func (m *NodeMgr) WriteNodesToRedis() {
	if !m.LoadNodesFromFile("/tmp/allnodes.json") {
		return
	}
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

package nodemgr

import (
	"fmt"
	"mikutool/config"

	commonModel "github.com/qbox/mikud-live/common/model"
)

type NodeMgr struct {
	allRootNodesMapByNodeId map[string]*commonModel.RtNode
	allNodesMap             map[string]*commonModel.RtNode
	conf                    *config.Config
}

func NewNodeMgr() *NodeMgr {
	return &NodeMgr{
		allRootNodesMapByNodeId: make(map[string]*commonModel.RtNode),
		allNodesMap:             make(map[string]*commonModel.RtNode),
	}
}

func (m *NodeMgr) SetConf(conf *config.Config) {
	m.conf = conf
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

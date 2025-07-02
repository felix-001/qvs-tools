package nodemgr

import (
	commonModel "github.com/qbox/mikud-live/common/model"
)

type NodeMgr struct {
	allRootNodesMapByNodeId map[string]*commonModel.RtNode
	allNodesMap             map[string]*commonModel.RtNode
}

func NewNodeMgr() *NodeMgr {
	return &NodeMgr{
		allRootNodesMapByNodeId: make(map[string]*commonModel.RtNode),
		allNodesMap:             make(map[string]*commonModel.RtNode),
	}
}

func (m *NodeMgr) GetRootNodeByNodeId(nodeId string) *commonModel.RtNode {
	return m.allRootNodesMapByNodeId[nodeId]
}

func (m *NodeMgr) GetNodeByNodeId(nodeId string) *commonModel.RtNode {
	return m.allNodesMap[nodeId]
}

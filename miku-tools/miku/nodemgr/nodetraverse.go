package nodemgr

import (
	"fmt"

	public "github.com/qbox/mikud-live/common/model"
)

type NodeCallback interface {
	OnIp(node *public.RtNode, ip *public.RtIpStatus)
	OnNode(node *public.RtNode)
	Done(result map[string]int)
}

func (m *NodeMgr) Register(module NodeCallback) {
	m.modules = append(m.modules, module)
}

func (m *NodeMgr) GetMoudleCnt() int {
	return len(m.modules)
}

func (m *NodeMgr) Traverse() {
	fmt.Println("NodeTraverse")
	allNodes := m.allNodesMap
	result := map[string]int{}
	for _, node := range allNodes {
		result["totalNodes"]++
		if !m.filterMgr.FilterNode(node) {
			continue
		}
		for _, module := range m.modules {
			module.OnNode(node)
		}
		for _, ip := range node.Ips {
			result["totalIps"]++
			if !m.filterMgr.FilterIp(&ip) {
				continue
			}
			for _, module := range m.modules {
				module.OnIp(node, &ip)
			}
		}
	}

	for _, module := range m.modules {
		module.Done(result)
	}
}

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
	m.filterMgr.LoadFilterData()
	allNodes := m.allNodesMap
	totalNodes := 0
	totalIps := 0
	for _, node := range allNodes {
		totalNodes++
		if !m.filterMgr.FilterNodeByAvailability(node) {
			continue
		}
		for _, module := range m.modules {
			module.OnNode(node)
		}
		for _, ip := range node.Ips {
			totalIps++
			if !m.filterMgr.FilterIpByAvailability(node, &ip) {
				continue
			}
			for _, module := range m.modules {
				module.OnIp(node, &ip)
			}
		}
	}

	filterResult := m.filterMgr.GetStatistics()
	filterResult["totalNodes"] = totalNodes
	filterResult["totalIps"] = totalIps

	for _, module := range m.modules {
		module.Done(filterResult)
	}
}

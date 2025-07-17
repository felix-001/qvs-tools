package nodemgr

import (
	"fmt"

	public "github.com/qbox/mikud-live/common/model"
	commonUtil "github.com/qbox/mikud-live/common/util"
)

type NodeCallback interface {
	OnIp(node *public.RtNode, ip *public.RtIpStatus)
	OnNode(node *public.RtNode)
	Done(result map[string]int)
}

func IpFilterPrivate(ip *public.RtIpStatus) (string, bool) {
	return "PrivateIp", !commonUtil.IsPrivateIP(ip.Ip)
}

func IpFilterForbidden(ip *public.RtIpStatus) (string, bool) {
	return "IpFrobidden", !ip.Forbidden
}

func IpFilterProbeSpeed(ip *public.RtIpStatus) (string, bool) {
	if ip.IPStreamProbe.Speed > 0 && ip.IPStreamProbe.MinSpeed > 0 &&
		ip.IPStreamProbe.Speed < 8 &&
		ip.IPStreamProbe.MinSpeed < 6 {
		return "ProbeSpeed", false
	}
	return "ProbeSpeed", true
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

package nodemgr

import (
	"fmt"

	public "github.com/qbox/mikud-live/common/model"
	commonUtil "github.com/qbox/mikud-live/common/util"
)

type NodeFilter func(node *public.RtNode) (string, bool)
type IpFilter func(ip *public.RtIpStatus) (string, bool)

type NodeCallback interface {
	OnIp(node *public.RtNode, ip *public.RtIpStatus)
	OnNode(node *public.RtNode)
	Done(result map[string]int)
	GetNodeFilters() []NodeFilter
	GetIpFilters() []IpFilter
}

var DefaultNodeFilters = []NodeFilter{
	NodeFilterDynamic,
	NodeFilterServing,
	NodeFilterNoStreamdPorts,
	NodeFilterNotBanProv,
	NodeFilterTimeLimit,
}

func NodeFilterDynamic(node *public.RtNode) (string, bool) {
	return "NotDynamic", node.IsDynamic
}

func NodeFilterNotBanProv(node *public.RtNode) (string, bool) {
	return "BanProv", !node.IsBanTransProv
}

func NodeFilterServing(node *public.RtNode) (string, bool) {
	return "NotServing", node.RuntimeStatus == "Serving"
}

func NodeFilterNoStreamdPorts(node *public.RtNode) (string, bool) {
	if node.StreamdPorts.Http == 0 {
		return "NoStreamdPorts", false
	}
	if node.StreamdPorts.Wt == 0 {
		return "NoStreamdPorts", false
	}
	if node.StreamdPorts.Https == 0 {
		return "NoStreamdPorts", false
	}
	return "NoStreamdPorts", true
}

func NodeFilterTimeLimit(node *public.RtNode) (string, bool) {
	return "TimeLimit", len(node.Schedules) == 0
}

var DefaultIpFilters = []IpFilter{
	IpFilterPrivate,
	IpFilterForbidden,
	IpFilterProbeSpeed,
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
	result := make(map[string]int)
	for _, node := range allNodes {
		result["totalNodes"]++
		pass := true
		for _, module := range m.modules {
			filters := module.GetNodeFilters()
			for _, filter := range filters {
				if name, ok := filter(node); !ok {
					result[name]++
					pass = false
					break
				}
			}
			if !pass {
				break
			}
			module.OnNode(node)
		}
		if !pass {
			continue
		}
		for _, ip := range node.Ips {
			result["totalIps"]++
			for _, module := range m.modules {
				pass = true
				filters := module.GetIpFilters()
				for _, filter := range filters {
					if name, ok := filter(&ip); !ok {
						result[name]++
						pass = false
						break
					}
				}
				if !pass {
					break
				}

				module.OnIp(node, &ip)
			}
		}
	}

	for _, module := range m.modules {
		module.Done(result)
	}
}

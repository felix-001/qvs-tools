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
	NodeFilterNotBanProv,
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

var DefaultIpFilters = []IpFilter{
	IpFilterPrivate,
}

func IpFilterPrivate(ip *public.RtIpStatus) (string, bool) {
	return "PrivateIp", !commonUtil.IsPrivateIP(ip.Ip)
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

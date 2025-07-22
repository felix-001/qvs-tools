package nodemgr

import (
	"fmt"

	commonModel "github.com/qbox/mikud-live/common/model"
)

type Ipv6Getter struct {
}

func (b *Ipv6Getter) OnNode(node *commonModel.RtNode) {
}

func (b *Ipv6Getter) OnIp(node *commonModel.RtNode, ip *commonModel.RtIpStatus) {
	if ip.IsIPv6 {
		fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "ip:", ip.Ip)
	}
}

func (b *Ipv6Getter) Done(result map[string]int) {
}

func (m *NodeMgr) GetIpv6Nodes() {
	g := &Ipv6Getter{}
	m.Register(g)
	m.filterMgr.SetFilterType(FilterTypeMongo)
	m.Traverse()
}

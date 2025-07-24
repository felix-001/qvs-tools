package nodemgr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	commonModel "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
)

type Ipv6Getter struct {
}

func (b *Ipv6Getter) OnNode(node *commonModel.RtNode) {
}

func (b *Ipv6Getter) OnIp(node *commonModel.RtNode, ip *commonModel.RtIpStatus) {
	fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "ip:", ip.Ip)
}

func (b *Ipv6Getter) Done(result map[string]int) {
	log.Printf("%+v\n", result)
}

func (m *NodeMgr) GetIpv6Nodes() {
	g := &Ipv6Getter{}
	m.Register(g)
	m.filterMgr.SetFilterType(FilterTypeLocal)
	m.filterMgr.FilterSwitch("Dynamic", false)
	m.filterMgr.FilterSwitch("Static", true)
	m.filterMgr.FilterSwitch("Abilities", false)
	m.filterMgr.FilterSwitch("Serving", false)
	m.filterMgr.FilterSwitch("Services", false)
	m.filterMgr.FilterSwitch("Ipv6", true)
	m.Traverse()
}

func (m *NodeMgr) GetIpv6DnsRecords() {
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := m.resources.DnsPodCli.GetRecords(xl, m.conf.Domain, "", "", 0)
	if err != nil {
		log.Println(err)
		return
	}
	if m.conf.Detail {
		bytes, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(bytes))
	}
	for _, record := range resp.RecordList {
		if record.Name == nil {
			continue
		}
		if m.conf.Name != "" && *record.Name != m.conf.Name {
			continue
		}
		if record.Line == nil {
			continue
		}
		if record.Value == nil {
			continue
		}
		if record.Type == nil {
			continue
		}
		if *record.Type != "AAAA" {
			continue
		}
		if record.Status == nil {
			continue
		}
		if *record.Status != "ENABLE" {
			continue
		}
		remark := ""
		if record.Remark != nil {
			remark = *record.Remark
		}
		m.conf.Ip = *record.Value
		nodeId := m.GetNodeByIp()
		fmt.Println("name:", *record.Name, "line:", *record.Line, "value:", *record.Value, "remark:", remark, "nodeId:", nodeId)
	}

}

package nodemgr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mikutool/public/util"

	commonModel "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
	"github.com/qbox/pili/common/ipdb.v1"
)

type Ipv6Getter struct {
	nodeMap  map[string]int
	areaMap  map[string]int
	provMap  map[string]int
	ipParser *ipdb.City
}

func (b *Ipv6Getter) OnNode(node *commonModel.RtNode) {
}

func (b *Ipv6Getter) OnIp(node *commonModel.RtNode, ip *commonModel.RtIpStatus) {
	fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "ip:", ip.Ip, "isp:", ip.Isp)
	_, _, area, prov := util.GetLocate(ip.Ip, b.ipParser)
	b.areaMap[area]++
	b.provMap[prov]++
	b.nodeMap[node.Id]++
}

func (b *Ipv6Getter) Done(result map[string]int) {
	log.Printf("%+v ipv6 nodes count: %d\n", result, len(b.nodeMap))
	for k, v := range b.areaMap {
		fmt.Println(k+":", v)
	}
	for k, v := range b.provMap {
		fmt.Println(k+":", v)
	}
}

func (m *NodeMgr) GetIpv6Nodes() {
	g := &Ipv6Getter{
		nodeMap:  make(map[string]int),
		areaMap:  make(map[string]int),
		provMap:  make(map[string]int),
		ipParser: m.resources.IpParser}
	m.Register(g)
	m.filterMgr.SetFilterType(FilterTypeLocal)
	m.filterMgr.FilterSwitch("Dynamic", true)
	m.filterMgr.FilterSwitch("Static", false)
	m.filterMgr.FilterSwitch("Abilities", true)
	m.filterMgr.FilterSwitch("Serving", true)
	m.filterMgr.FilterSwitch("Services", true)
	m.filterMgr.FilterSwitch("Ipv6", true)
	m.Traverse()
}

func (m *NodeMgr) GetIpv6DnsRecords() {
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := m.resources.DnsPodCli.GetRecords(xl, m.conf.Domain, m.conf.Host, "", 0)
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
		//if *record.Type != "AAAA" {
		//continue
		//}
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
		//nodeId := m.GetNodeByIp()
		fmt.Println("name:", *record.Name, "line:", *record.Line, "value:", *record.Value, "remark:", remark)
	}

}

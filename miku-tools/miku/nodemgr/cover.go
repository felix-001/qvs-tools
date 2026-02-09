package nodemgr

import (
	"fmt"
	"log"
	"mikutool/public/util"

	commonModel "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
)

type CoverStatistics struct {
	ipParser   *ipdb.City
	ispAreaMap map[string]int
	nodeCntMap map[string]int
	maxBwMap   map[string]float64
	nodeCnt    int
}

func (b *CoverStatistics) OnNode(node *commonModel.RtNode) {
	_, isp, _, _ := util.GetLocate(node.Ips[0].Ip, b.ipParser)
	if isp != "" {
		b.nodeCntMap[isp]++
	}
	b.nodeCnt++
}

func (b *CoverStatistics) OnIp(node *commonModel.RtNode, ip *commonModel.RtIpStatus) {
	_, isp, area, _ := util.GetLocate(ip.Ip, b.ipParser)
	if isp == "" {
		return
	}
	b.ispAreaMap[area+isp]++
	b.maxBwMap[isp] += ip.MaxOutMBps
}

func (b *CoverStatistics) Done(result map[string]int) {
	log.Println("total:", len(b.ispAreaMap))
	log.Println("ip个数:")
	for k, v := range b.ispAreaMap {
		fmt.Println("\t"+k+":", v)
	}
	for _, isp := range util.Isps {
		for _, area := range util.Areas {
			if _, ok := b.ispAreaMap[area+isp]; !ok {
				fmt.Println(area+isp, "not cover")
			}
		}
	}
	log.Println("node个数:")
	for k, v := range b.nodeCntMap {
		fmt.Println("\t"+k, ":", v)
	}
	log.Println("建设带宽:")
	for k, v := range b.maxBwMap {
		fmt.Printf("\t%s : %.1fGbps\n", k, v*8/1000)
	}
	for k, v := range result {
		fmt.Println(k, ":", v)
	}
	log.Println("nodeCnt:", b.nodeCnt)
}

type TimeLimit struct {
	ipParser *ipdb.City
	nodeCnt  int
	ipCnt    int
	bw       float64
}

func (t *TimeLimit) OnNode(node *commonModel.RtNode) {
	t.nodeCnt++
	log.Println("node:", node.Id)
}

func (t *TimeLimit) OnIp(node *commonModel.RtNode, ip *commonModel.RtIpStatus) {
	t.ipCnt++
	t.bw += ip.MaxOutMBps
}

func (t *TimeLimit) Done(result map[string]int) {
	log.Println("time limit nodeCnt:", t.nodeCnt, "ipCnt:", t.ipCnt, "bw:", t.bw*8/1000)
	for k, v := range result {
		fmt.Println("\t"+k, ":", v)
	}
}

func (m *NodeMgr) NodesCover() {
	b := &CoverStatistics{
		ipParser:   m.resources.IpParser,
		ispAreaMap: make(map[string]int),
		nodeCntMap: make(map[string]int),
		maxBwMap:   make(map[string]float64),
	}
	m.Register(b)
	m.Traverse()

	log.Println("dump time limit nodes:")
	timelimit := &TimeLimit{
		ipParser: m.resources.IpParser,
	}
	m.filterMgr.SetFilterType(FilterTypeLocal)
	m.filterMgr.SwitchDefault()
	m.filterMgr.FilterSwitch("NotBanProv", false)
	m.filterMgr.FilterSwitch("Ipv6", false)
	m.filterMgr.FilterSwitch("Static", false)
	m.filterMgr.FilterSwitch("AvailableBw", false)
	m.filterMgr.FilterSwitch("Serving", false)
	m.filterMgr.FilterSwitch("TimeLimit", false)
	m.filterMgr.FilterSwitch("NotNat1", false)
	m.filterMgr.FilterSwitch("Abilities", false)
	m.filterMgr.FilterSwitch("Services", false)
	m.filterMgr.FilterSwitch("NoStreamdPorts", false)
	m.Register(timelimit)
	m.Traverse()

	m.Test()
}

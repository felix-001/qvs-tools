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
}

func (b *CoverStatistics) OnNode(node *commonModel.RtNode) {
}

func (b *CoverStatistics) OnIp(node *commonModel.RtNode, ip *commonModel.RtIpStatus) {
	isp, area, _ := util.GetLocate(ip.Ip, b.ipParser)
	if isp != "" {
		b.ispAreaMap[area+isp]++
	}

}

func (b *CoverStatistics) Done(result map[string]int) {
	log.Println("total:", len(b.ispAreaMap))
	for k, v := range b.ispAreaMap {
		fmt.Println(k+":", v)
	}
	for _, isp := range util.Isps {
		for _, area := range util.Areas {
			if _, ok := b.ispAreaMap[area+isp]; !ok {
				fmt.Println(area+isp, "not cover")
			}
		}
	}
}

func (m *NodeMgr) NodesCover() {
	b := &CoverStatistics{
		ipParser:   m.resources.IpParser,
		ispAreaMap: make(map[string]int),
	}
	m.Register(b)
	m.Traverse()
}

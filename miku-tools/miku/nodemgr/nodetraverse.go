package nodemgr

import (
	"log"

	public "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/mikud-live/common/util"
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

func (m *NodeMgr) Test() {
	cnt := 0
	var bw float64
	bwMap := make(map[string]float64)
	for _, node := range m.allNodesMap {
		if len(node.Schedules) == 0 {
			//log.Println("not time limit node")
			continue
		}
		if !node.IsDynamic {
			continue
		}
		if node.RuntimeStatus != "Serving" {
			continue
		}
		ability, ok := node.Abilities["live"]
		if !ok || !ability.Can || ability.Frozen {
			continue
		}
		for _, schedule := range node.Schedules {
			if len(schedule.ScheduleISPs) > 0 {
				continue
			}
			if schedule.ScheduledStart == 0 && schedule.ScheduledEnd == 86400 {
				log.Println("not time limit node")
				continue
			}
		}
		for _, ipInfo := range node.Ips {
			if util.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			bw += ipInfo.MaxOutMBps
			if ipInfo.Isp == "" {
				log.Println("ipInfo.Isp is empty")
			}
			bwMap[ipInfo.Isp] += ipInfo.MaxOutMBps * 8 / 1000
		}
		cnt++
	}
	log.Println("cnt:", cnt, "bw:", bw*8/1000)
	log.Println("bwMap:", bwMap)
}

func (m *NodeMgr) Traverse() {
	log.Println("NodeTraverse")
	m.filterMgr.LoadFilterData()
	allNodes := m.allNodesMap
	totalNodes := 0
	totalIps := 0
	for _, node := range allNodes {
		totalNodes++
		if !m.filterMgr.FilterNode(node) {
			continue
		}
		for _, module := range m.modules {
			module.OnNode(node)
		}
		for _, ip := range node.Ips {
			totalIps++
			if !m.filterMgr.FilterIp(node, &ip) {
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

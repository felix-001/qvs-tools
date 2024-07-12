package main

import (
	"log"

	publicUtil "github.com/qbox/mikud-live/common/util"
)

// 统计节点有多个网卡的情况，是否有跨运营商

func (s *Parser) nodesNetinterfaceStatistics() {
	nodeHaveMoreThan1InterfaceCnt := 0
	notServingCnt := 0
	for _, node := range s.allNodesMap {
		if !node.IsDynamic {
			continue
		}
		if node.RuntimeStatus != "Serving" {
			notServingCnt++
		}
		lastIsp := ""
		interfaceCnt := 0
		for _, ipInfo := range node.Ips {
			if ipInfo.IsIPv6 {
				continue
			}
			if lastIsp != "" && ipInfo.Isp != "" && lastIsp != ipInfo.Isp {
				log.Println("node:", node.Id, "lastIsp:", lastIsp, "isp:", ipInfo.Isp, "machineId:", node.MachineId, "idc:", node.Idc, "ip:", ipInfo.Ip)
			}
			lastIsp = ipInfo.Isp
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				//log.Println("private ip", ipInfo.Ip)
				continue
			}
			if ipInfo.Ip == "" {
				log.Println("node", node.Id, "ip empty")
				continue
			}
			interfaceCnt++
		}
		if interfaceCnt > 1 {
			nodeHaveMoreThan1InterfaceCnt++
		}
	}
	log.Println("拥有多张网卡的节点个数:", nodeHaveMoreThan1InterfaceCnt, "总共节点个数:", len(s.allNodesMap), "notServingCnt:", notServingCnt)
}

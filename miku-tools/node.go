package main

import "fmt"

func (s *Parser) GetNodeByIp() {
	for _, node := range s.allNodesMap {
		for _, ipInfo := range node.Ips {
			if ipInfo.Ip == s.conf.Ip {
				_, ok := s.allRootNodesMapByNodeId[node.Id]
				fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "isRoot:", ok)
				break
			}
		}
	}
}

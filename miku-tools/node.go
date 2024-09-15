package main

import "fmt"

func (s *Parser) GetNodeByIp() {
	for _, node := range s.allNodesMap {
		for _, ipInfo := range node.Ips {
			if ipInfo.Ip == s.conf.Ip {
				fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId)
				break
			}
		}
	}
}

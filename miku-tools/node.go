package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	zlog "github.com/rs/zerolog/log"
)

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

func (s *Parser) NodeDis() {
	areaMap := make(map[string]bool)
	for _, node := range s.allNodesMap {
		if node.RuntimeStatus != "Serving" {
			continue
		}
		if node.StreamdPorts.Http == 0 {
			continue
		}
		if node.StreamdPorts.Rtmp == 0 {
			continue
		}
		if !util.CheckNodeUsable(zlog.Logger, node, consts.TypeLive) {
			continue
		}
		isp, area, _ := getNodeLocate(node, s.IpParser)
		areaMap[area+isp] = true

	}

	needAreas := make([]string, 0)
	for _, area := range Areas {
		for _, isp := range Isps {
			areaIsp := area + isp
			if _, ok := areaMap[areaIsp]; !ok {
				needAreas = append(needAreas, areaIsp)
			}
		}
	}
	bytes, err := json.MarshalIndent(areaMap, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
	fmt.Println("needAreas:", needAreas)
}

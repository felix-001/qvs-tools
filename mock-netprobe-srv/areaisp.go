package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
)

func (s *NetprobeSrv) DumpAreaIsp(paramMap map[string]string) string {
	areaIspMap := map[string]int{}
	for _, node := range s.nodes {
		for _, ip := range node.Ips {
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			areaIsp := s.getIpAreaIsp(ip.Ip)
			areaIspMap[areaIsp] += 1
			break
		}
	}
	jsonbody, err := json.Marshal(areaIspMap)
	if err != nil {
		log.Println(err)
	}
	return string(jsonbody)
}

func (s *NetprobeSrv) getIpAreaIsp(ip string) string {
	locate, err := s.ipParser.Find(ip)
	if err != nil {
		log.Println("get locate of ip", ip, "err", err)
		return ""
	}
	areaIpsKey, _ := util.GetAreaIspKey(locate)
	areaIsp := strings.TrimPrefix(areaIpsKey, util.AreaIspKeyPrefix)
	return areaIsp
}

func (s *NetprobeSrv) GetAreaIspNodesInfo(needAreaIsp string) []*model.RtNode {
	areaIspGroup := make(map[string][]*model.RtNode)
	for _, node := range s.nodes {
		for _, ip := range node.Ips {
			locate, err := s.ipParser.Find(ip.Ip)
			if err != nil {
				log.Println("get locate of ip", ip.Ip, "err", err)
				continue
			}
			areaIpsKey, _ := util.GetAreaIspKey(locate)
			areaIsp := strings.TrimPrefix(areaIpsKey, util.AreaIspKeyPrefix)
			areaIspGroup[areaIsp] = append(areaIspGroup[areaIsp], node)
			break
		}
	}
	log.Println("node count:", len(areaIspGroup[needAreaIsp]))
	return areaIspGroup[needAreaIsp]
}

func (s *NetprobeSrv) GetAreaInfo(areaIsp string) string {
	info := map[string]float64{}
	for _, node := range s.nodes {
		for _, ip := range node.Ips {
			if ip.IsIPv6 {
				continue
			}
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			locate, err := s.ipParser.Find(ip.Ip)
			if err != nil {
				log.Println("get locate of ip", ip.Ip, "err", err)
				continue
			}
			areaIpsKey, _ := util.GetAreaIspKey(locate)
			areaIsp_ := strings.TrimPrefix(areaIpsKey, util.AreaIspKeyPrefix)
			if areaIsp != areaIsp_ {
				continue
			}
			info[node.Id] += ip.MaxOutMBps
		}
	}
	pairs := SortFloatMap(info)
	jsonbody, err := json.Marshal(pairs)
	if err != nil {
		log.Println(err)
	}
	return string(jsonbody)
}

package main

import (
	"log"

	"github.com/qbox/mikud-live/common/util"
)

func (s *Parser) PcdnDbg() {
	provinceIpMap := make(map[string]map[string]string) // key1: isp, key2: province, value: ip
	for _, node := range s.allNodesMap {
		for _, ipInfo := range node.Ips {
			if util.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			if ipInfo.IsIPv6 {
				continue
			}
			isp, _, province := getLocate(ipInfo.Ip, s.ipParser)
			if province == "" {
				continue
			}
			if _, ok := provinceIpMap[isp]; !ok {
				provinceIpMap[isp] = make(map[string]string)
			}
			provinceIpMap[isp][province] = ipInfo.Ip
		}
	}
	log.Println("province ip map cnt: ", len(provinceIpMap))
	for isp, data := range provinceIpMap {
		log.Println(isp, len(data))
	}
}

package main

import publicUtil "github.com/qbox/mikud-live/common/util"

// 临时的代码放到这里

func (s *Parser) Staging() {
	switch s.conf.SubCmd {
	case "getpcdn":
		s.getPcdnFromSchedAPI(true, false)
	case "volc":
		s.fetchVolcOriginUrl()
	case "ipv6":
		s.dumpNodeIpv4v6Dis()
	}
}

type NodeDis struct {
	Ipv4Cnt int
	Ipv6Cnt int
}

func (s *Parser) dumpNodeIpv4v6Dis() {
	areaIpv4v6CntMap := make(map[string]*NodeDis)
	for _, node := range s.allNodesMap {
		for _, ip := range node.Ips {
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			if !IsPublicIPAddress(ip.Ip) {
				continue
			}
			isp, area, _ := getLocate(ip.Ip, s.IpParser)
			if area == "" {
				continue
			}
			key := area + "_" + isp
			m := areaIpv4v6CntMap[key]
			if m == nil {
				m = &NodeDis{}
				areaIpv4v6CntMap[key] = m
			}
			if ip.IsIPv6 {
				m.Ipv6Cnt++
			} else {
				m.Ipv4Cnt++
			}
		}
	}

	areaIpv6PercentMap := make(map[string]int)
	for areaIsp, nodeDis := range areaIpv4v6CntMap {
		areaIpv6PercentMap[areaIsp] = nodeDis.Ipv6Cnt * 100 / (nodeDis.Ipv4Cnt + nodeDis.Ipv6Cnt)
	}
	pairs := SortIntMap(areaIpv6PercentMap)
	DumpSlice(pairs)

}

package main

import (
	"io/ioutil"
	"log"
	"net"
	"strings"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
)

func (s *Parser) DnsChk() {
	bytes, err := ioutil.ReadFile(s.conf.DnsResFile)
	if err != nil {
		log.Println("read fail", s.conf.DnsResFile, err)
		return
	}
	lines := strings.Split(string(bytes), "\r\n")
	for _, line := range lines[1:] {
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			log.Println("parse line err", line)
			continue
		}
		provinceIsp := parts[0]
		result := strings.Trim(parts[3], "\"")
		prov := ""
		for _, province := range Provinces {
			if strings.Contains(provinceIsp, province) {
				prov = province
				break
			}
		}
		if prov == "" {
			log.Println("dummy province:", provinceIsp)
			continue
		}
		area := monitorUtil.ProvinceAreaRelation(prov)
		if area == "" {
			log.Println("area empty", prov, provinceIsp)
			continue
		}
		isp := ""
		for _, _isp := range Isps {
			if strings.Contains(provinceIsp, isp) {
				isp = _isp
			}
		}
		if isp == "" {
			log.Println("dummy isp:", provinceIsp)
			continue
		}
		ips := strings.Split(result, "\n")
		validIp := ""
		for _, ip := range ips {
			if net.ParseIP(ip) != nil {
				validIp = ip
				break
			}
		}
		if validIp == "" {
			log.Println("no valid ip", result)
			continue
		}
		areaResult, ispResult, err := getIpAreaIsp(s.ipParser, validIp)
		if err != nil {
			log.Println("getIpAreaIsp err", validIp, err)
			continue
		}
		if areaResult != area {
			log.Println("area not same", areaResult, area)
		}
		if ispResult != isp {
			log.Println("isp not same", ispResult, isp)
		}
	}
}

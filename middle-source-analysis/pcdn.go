package main

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/qbox/mikud-live/common/util"
)

type PlaycheckReq struct {
	Bucket   string            `json:"bucket"`
	Key      string            `json:"key"`
	Url      string            `json:"url"`
	Remote   string            `json:"remoteAddr"`
	Local    string            `json:"localAddr"`
	Node     string            `json:"nodeId"`
	ConnId   string            `json:"connectId"`
	Master   string            `json:"master"`
	Protocol string            `json:"protocol"` // 协议类型，非必填，目前仅hls协议传递
	Headers  map[string]string `json:"headers"`
	User     string            `json:"user"`
}

type PlayCheckResp struct {
	Uid        uint32 `json:"uid"`
	ErrCode    string `json:"code"`
	Message    string `json:"message"`
	ConnectId  string `json:"connectId"`
	Url302     string `json:"redirectUrl"`
	RewriteUrl string `json:"rewriteUrl"`
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	RegTsM3u8  string `json:"regTsM3u8"`  // m3u8文件中ts格式，替换其中的${app}/${stream}/${file}
	FlowMethod int    `json:"flowMethod"` // 计量方式: 1: miku计量系统; 2: pili计量系统; 其它值miku&pili计量系统
}

func (s *Parser) playcheck(ip string) *PlayCheckResp {
	req := PlaycheckReq{
		Bucket: "douyuflv",
		Key:    "4549169raUKQVzf4_900",
		Url:    "http://qn3.douyucdn.cn/live/1226741r3fmmEOb7.flv?did=a75e6982-7538-4629-ad3c-fd0d60b1ba54&expire=0&ip=120.230.111.210&isp=&logo=0&mcid2=0&mix=0&origin=tct&pt=1&sid=397423057&st=0&token=app-androidxlv-0-1226741-0dc52b22d029568980d4d39a0dd754645d73804f0aa4d875&um=0&ver=2.6.1&wsAuth=dda22a267d16cece907605bb44c23f37",
		Node:   "2b8f0c5a-85d0-3c4a-bbd8-ac77a82d607b-rtc-gdfsh-dls-1-7",
		Remote: ip,
		ConnId: "connetId",
	}
	bytes, err := json.Marshal(&req)
	if err != nil {
		log.Println(err)
		return nil
	}
	var resp PlayCheckResp
	addr := "http://10.34.146.62:6060/api/v1/playcheck"
	if err := s.post(addr, string(bytes), &resp); err != nil {
		log.Println(err)
		return nil
	}
	return &resp
}

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
			if ipInfo.Ip == "" {
				log.Println("ip empty")
				continue
			}
			isp, _, province := getLocate(ipInfo.Ip, s.ipParser)
			if province == "" {
				continue
			}
			if isp != "联通" && isp != "电信" && isp != "移动" {
				continue
			}
			if _, ok := provinceIpMap[isp]; !ok {
				provinceIpMap[isp] = make(map[string]string)
			}
			provinceIpMap[isp][province] = ipInfo.Ip
		}
	}
	log.Println("province ip map cnt: ", len(provinceIpMap))
	cnt := 0
	totalCnt := 0
	ipV6Cnt := 0
	for isp, data := range provinceIpMap {
		for province, ip := range data {
			if ip == "" {
				log.Println("ip empty")
				continue
			}
			resp := s.playcheck(ip + ":8080")
			u, err := url.Parse(resp.Url302)
			if err != nil {
				log.Println(err, resp.Url302)
				continue
			}
			if IsIpv6(u.Hostname()) {
				ipV6Cnt++
			}
			totalCnt++
			//log.Println("redirectUr:", resp.Url302)
			nodeIsp, _, nodeProvince := getLocate(u.Hostname(), s.ipParser)
			if nodeIsp != isp {
				log.Println("isp not match, ", "isp:", isp, "nodeIsp:",
					nodeIsp, "ip:", ip, "nodeIp:", u.Hostname(), "province:", province,
					"nodeProvince:", nodeProvince)
				cnt++
				continue
			}
			if province != nodeProvince {
				log.Println("province not match, ", "province:", province,
					"nodeProvince:", nodeProvince, "ip:", ip, "nodeIp:",
					u.Hostname(), "isp:", isp, "nodeIsp:",
					nodeIsp)
				cnt++
				continue
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
	log.Println("err cnt:", cnt, "ipv6 cnt:", ipV6Cnt, "totalCnt:", totalCnt)
}

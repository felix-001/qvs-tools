package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"
	"time"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
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

func (s *Parser) fetchProvincesIps() {
	provinceIpMap := make(map[string]map[string]string) // key1: isp, key2: province, value: ip
	provinces := []string{"湖南", "内蒙古", "贵州", "山西", "河南", "天津", "江苏", "四川", "西藏", "湖北", "上海", "江西", "广东", "陕西", "辽宁", "河北", "山东", "福建", "云南", "新疆", "黑龙江", "宁夏", "安徽", "重庆", "浙江", "吉林", "海南", "甘肃", "青海", "北京", "广西"}
	log.Println("len(provinces)", len(provinces))
	isps := []string{"移动", "电信", "联通"}
	for _, province := range provinces {
		for _, isp := range isps {
			ip := s.GetIpByProvinceIsp(province, isp)
			if ip == "" {
				log.Println(province, isp, "ip empty")
				continue
			}
			if _, ok := provinceIpMap[isp]; !ok {
				provinceIpMap[isp] = make(map[string]string)
			}
			provinceIpMap[isp][province] = ip
		}
	}
	jsonbody, err := json.Marshal(provinceIpMap)
	if err != nil {
		log.Println(err)
		return
	}
	err = ioutil.WriteFile("ips.json", jsonbody, 0644)
	if err != nil {
		log.Println(err)
		return
	}
}

func (s *Parser) loadTestIpData() map[string]map[string]string {
	bytes, err := ioutil.ReadFile("ips.json")
	if err != nil {
		log.Println("read fail", "ips.json", err)
		return nil
	}
	provinceIpMap := make(map[string]map[string]string) // key1: isp, key2: province, value: ip
	if err := json.Unmarshal(bytes, &provinceIpMap); err != nil {
		log.Println(err)
		return nil
	}
	return provinceIpMap
}

func (s *Parser) PcdnDbg() {
	provinceIpMap := s.loadTestIpData()

	cnt := 0
	totalCnt := 0
	ipV6Cnt := 0
	areaErrCnt := 0
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
			area := monitorUtil.ProvinceAreaRelation(province)
			nodeArea := monitorUtil.ProvinceAreaRelation(nodeProvince)
			if area != nodeArea {
				areaErrCnt++
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
	log.Println("err cnt:", cnt, "ipv6 cnt:", ipV6Cnt, "totalCnt:", totalCnt, "areaErrCnt:", areaErrCnt)
}

func (s *Parser) Pcdn() {

}

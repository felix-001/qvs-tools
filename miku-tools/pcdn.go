package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
	schedUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
	schedModel "github.com/qbox/mikud-live/cmd/sched/model"
	"github.com/qbox/mikud-live/common/model"
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
	bytes, err := ioutil.ReadFile("/tmp/ips.json")
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

func (s *Parser) getPcdn() string {
	ispProvincesIpMap := s.loadTestIpData()
	ip := ispProvincesIpMap[s.conf.Isp][s.conf.Province]
	if ip == "" {
		s.logger.Info().Str("isp", s.conf.Isp).Str("province", s.conf.Province).Msg("no ip found")
		return ""
	}
	host := s.conf.Bucket
	if s.conf.Bucket == "dycold" {
		s.conf.Bucket = "miku-lived-douyu.qiniuapi.com"
	}
	addr := fmt.Sprintf("http://10.34.146.62:6060/%s/%s/douyugetpcdn?clientIp=%s&scheme=http&did=dummy&host=%s",
		s.conf.Bucket, s.conf.Stream, ip, host)

	data, err := get(addr)
	if err != nil {
		s.logger.Info().Err(err).Str("addr", addr).Msg("req douyugetpcdn err")
		return ""
	}
	fmt.Println(data)
	var resp model.DouyuPcdnResp
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		log.Println(err)
		return ""
	}
	return resp.PCDN
}

func (s *Parser) Pcdn() {
	pcdn := s.getPcdn()
	fmt.Println(pcdn)
}

func (s *Parser) getPcdnFromSchedAPI(skipReport, skipRoot bool) (string, string) {
	addr := "http://10.34.146.62:6060/api/v1/nodes?level=default&dimension=area&mode=detail&ipversion=ipv4"
	resp, err := get(addr)
	if err != nil {
		s.logger.Error().Err(err).Str("addr", addr).Msg("get nodes err")
		return "", ""
	}
	//fmt.Println(resp)
	areaNodesMap := make(map[string][]*schedModel.NodeIpsPair)
	if err := json.Unmarshal([]byte(resp), &areaNodesMap); err != nil {
		s.logger.Error().Err(err).Msg("unmashal err")
		return "", ""
	}
	key := fmt.Sprintf("area_isp_group_%s_%s", s.conf.Area, s.conf.Isp)
	nodes, ok := areaNodesMap[key]
	if !ok {
		s.logger.Error().
			Str("area", s.conf.Area).
			Str("isp", s.conf.Isp).
			Msg("area isp not found nodes")
		return "", ""
	}
	if len(nodes) == 0 {
		s.logger.Error().Msg("nodes len is 0")
		return "", ""
	}
	nodesMap := s.getNodesByStreamId()
	streamNodes := nodesMap[key]
	if streamNodes == nil {
		s.logger.Error().Str("key", key).Msg("not found stream nodes")
	}
	pcdn := ""
	var selectNode *schedModel.NodeIpsPair
	for _, node := range nodes {
		if skipReport {
			for _, detail := range streamNodes {
				if node.Node.Id == detail.NodeId {
					s.logger.Info().Str("node", node.Node.Id).Msg("skip node")
					continue
				}
			}
		}
		if skipRoot {
			if _, ok := s.allRootNodesMapByNodeId[node.Node.Id]; ok {
				s.logger.Info().Str("node", node.Node.Id).Msg("skip root node")
				continue
			}
		}
		for _, ipInfo := range node.Ips {
			if ipInfo.IsIPv6 {
				continue
			}
			if util.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			pcdn = fmt.Sprintf("%s:%d", ipInfo.Ip, node.Node.StreamdPorts.Http)
			selectNode = node
			break
		}
	}
	if pcdn == "" {
		s.logger.Error().Msg("pcdn empty")
		return "", ""
	}
	s.logger.Info().Str("nodeId", selectNode.Node.Id).Str("machineId", selectNode.Node.MachineId).Msg("selected node")
	return selectNode.Node.Id, pcdn
}

func (s *Parser) Pcdns() {
	totalAreaNotMatch := 0
	totalIspNotMatch := 0
	totalReqCnt := 0
	areaNotCoverCntMap := make(map[string]int)
	for _, province := range Provinces {
		for _, isp := range Isps {
			s.conf.Province = province
			s.conf.Isp = isp
			pcdn := s.getPcdn()
			parts := strings.Split(pcdn, ":")
			if len(parts) != 2 {
				return
			}
			pcdnIsp, pcdnArea, _ := getLocate(parts[0], s.ipParser)
			reqArea, _ := schedUtil.ProvinceAreaRelation(province)
			if reqArea != pcdnArea {
				s.logger.Error().Str("reqArea", reqArea).Str("pcdnArea", pcdnArea).Str("pcdn", pcdn).
					Str("reqIsp", isp).Str("pcdnIsp", pcdnIsp).Msg("area chk err")
				totalAreaNotMatch++
				areaNotCoverCntMap[reqArea+"_"+isp]++
			}
			if isp != pcdnIsp {
				s.logger.Error().Str("reqArea", reqArea).Str("pcdnArea", pcdnArea).Str("pcdn", pcdn).
					Str("reqIsp", isp).Str("pcdnIsp", pcdnIsp).Msg("isp chk err")
				totalIspNotMatch++
			}
			totalReqCnt++
		}
	}
	s.logger.Info().Int("totalAreaNotMatch", totalAreaNotMatch).Int("totalIspNotMatch", totalIspNotMatch).
		Int("areaNotMatch", len(areaNotCoverCntMap)).Int("totalReqCnt", totalReqCnt).Msg("Pcdns")
	for area, cnt := range areaNotCoverCntMap {
		s.logger.Info().Str("area", area).Int("cnt", cnt).Msg("area not match cnt")
	}
}

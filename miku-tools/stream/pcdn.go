package stream

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"middle-source-analysis/node"
	"middle-source-analysis/public"
	localUtil "middle-source-analysis/util"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
	schedUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
	schedModel "github.com/qbox/mikud-live/cmd/sched/model"
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/mikud-live/common/util"
	"github.com/rs/zerolog"
)

var sublogger = zerolog.New(os.Stdout).With().Timestamp().Logger()

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

func playcheck(ip string) *PlayCheckResp {
	scheme := "http"
	if Conf.Https {
		scheme += "s"
	}
	playUrl := fmt.Sprintf("%s://%s/%s/%%s?did=a75e6982-7538-4629-ad3c-fd0d60b1ba54&expire=0",
		scheme, Conf.Domain, Conf.Bucket, Conf.Stream, Conf.Format)
	req := PlaycheckReq{
		Bucket: Conf.Bucket,
		Key:    Conf.Stream,
		Url:    playUrl,
		Node:   Conf.Node,
		Remote: ip,
		ConnId: Conf.ConnId,
		User:   Conf.User,
	}
	bytes, err := json.Marshal(&req)
	if err != nil {
		log.Println(err)
		return nil
	}
	var resp PlayCheckResp
	addr := fmt.Sprintf("http://%s:6060/api/v1/playcheck", Conf.SchedIp)
	if err := localUtil.Post(addr, string(bytes), &resp); err != nil {
		log.Println(err)
		return nil
	}
	return &resp
}

func fetchProvincesIps() {
	provinceIpMap := make(map[string]map[string]string) // key1: isp, key2: province, value: ip
	provinces := []string{"湖南", "内蒙古", "贵州", "山西", "河南", "天津", "江苏", "四川", "西藏", "湖北", "上海", "江西", "广东", "陕西", "辽宁", "河北", "山东", "福建", "云南", "新疆", "黑龙江", "宁夏", "安徽", "重庆", "浙江", "吉林", "海南", "甘肃", "青海", "北京", "广西"}
	log.Println("len(provinces)", len(provinces))
	isps := []string{"移动", "电信", "联通"}
	for _, province := range provinces {
		for _, isp := range isps {
			ip := localUtil.GetIpByProvinceIsp(province, isp)
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
	err = ioutil.WriteFile("ipjson", jsonbody, 0644)
	if err != nil {
		log.Println(err)
		return
	}
}

func loadTestIpData() map[string]map[string]string {
	bytes, err := ioutil.ReadFile("/tmp/ipjson")
	if err != nil {
		log.Println("read fail", "ipjson", err)
		return nil
	}
	provinceIpMap := make(map[string]map[string]string) // key1: isp, key2: province, value: ip
	if err := json.Unmarshal(bytes, &provinceIpMap); err != nil {
		log.Println(err)
		return nil
	}
	return provinceIpMap
}

func LoopPlaycheck() {
	provinceIpMap := loadTestIpData()

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
			resp := playcheck(ip + ":8080")
			if resp == nil {
				continue
			}
			u, err := url.Parse(resp.Url302)
			if err != nil {
				log.Println(err, resp.Url302)
				continue
			}
			if localUtil.IsIpv6(u.Hostname()) {
				ipV6Cnt++
			}
			totalCnt++
			//log.Println("redirectUr:", resp.Url302)
			nodeIsp, _, nodeProvince := localUtil.GetLocate(u.Hostname(), IpParser)
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
	log.Println("err cnt:", cnt, "ipv6 cnt:", ipV6Cnt, "totalCnt:", totalCnt, "areaErrCnt:", areaErrCnt,
		"本省覆盖率:", (totalCnt-cnt)*100/totalCnt)
}

func getPcdn(did string) string {
	ispProvincesIpMap := loadTestIpData()
	ip := ispProvincesIpMap[Conf.Isp][Conf.Province]
	if ip == "" {
		return ""
	}
	host := Conf.Bucket
	if Conf.Bucket == "dycold" {
		Conf.Bucket = "miku-lived-douyu.qiniuapi.com"
	}
	scheme := "http"
	if Conf.Https {
		scheme += "s"
	}
	addr := fmt.Sprintf("http://10.34.146.62:6060/%s/%s/douyugetpcdn?clientIp=%s&scheme=%s&did=%s&host=%s&pcdn_error=%s",
		Conf.Bucket, Conf.Stream, ip, scheme, did, host, Conf.PcdnErr)

	data, err := localUtil.Get(addr)
	if err != nil {
		sublogger.Info().Err(err).Str("addr", addr).Msg("req douyugetpcdn err")
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

func Pcdn() {
	pcdn := getPcdn("dummy")
	fmt.Println(pcdn)
}

func getPcdnFromSchedAPI(skipReport, skipRoot bool) (string, string) {
	addr := "http://10.34.146.62:6060/api/v1/nodes?level=default&dimension=area&mode=detail&ipversion=ipv4"
	resp, err := localUtil.Get(addr)
	if err != nil {
		sublogger.Error().Err(err).Str("addr", addr).Msg("get nodes err")
		return "", ""
	}
	//fmt.Println(resp)
	areaNodesMap := make(map[string][]*schedModel.NodeIpsPair)
	if err := json.Unmarshal([]byte(resp), &areaNodesMap); err != nil {
		sublogger.Error().Err(err).Msg("unmashal err")
		return "", ""
	}
	key := fmt.Sprintf("area_isp_group_%s_%s", Conf.Area, Conf.Isp)
	nodes, ok := areaNodesMap[key]
	if !ok {
		sublogger.Error().
			Str("area", Conf.Area).
			Str("isp", Conf.Isp).
			Msg("area isp not found nodes")
		return "", ""
	}
	if len(nodes) == 0 {
		sublogger.Error().Msg("nodes len is 0")
		return "", ""
	}
	nodesMap := GetNodesByStreamId()
	streamNodes := nodesMap[key]
	if streamNodes == nil {
		sublogger.Error().Str("key", key).Msg("not found stream nodes")
	}
	pcdn := ""
	var selectNode *schedModel.NodeIpsPair
	for _, nodeInfo := range nodes {
		if skipReport {
			for _, detail := range streamNodes {
				if nodeInfo.Node.Id == detail.NodeId {
					sublogger.Info().Str("node", nodeInfo.Node.Id).Msg("skip node")
					continue
				}
			}
		}
		if skipRoot {
			if _, ok := node.AllRootNodesMapByNodeId[nodeInfo.Node.Id]; ok {
				sublogger.Info().Str("node", nodeInfo.Node.Id).Msg("skip root node")
				continue
			}
		}
		for _, ipInfo := range nodeInfo.Ips {
			if ipInfo.IsIPv6 {
				continue
			}
			if util.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			pcdn = fmt.Sprintf("%s:%d", ipInfo.Ip, nodeInfo.Node.StreamdPorts.Http)
			selectNode = nodeInfo
			break
		}
	}
	if pcdn == "" {
		sublogger.Error().Msg("pcdn empty")
		return "", ""
	}
	sublogger.Info().Str("nodeId", selectNode.Node.Id).Str("machineId", selectNode.Node.MachineId).Msg("selected node")
	return selectNode.Node.Id, pcdn
}

func Pcdns() {
	totalAreaNotMatch := 0
	totalIspNotMatch := 0
	totalReqCnt := 0
	areaNotCoverCntMap := make(map[string]int)
	for _, province := range public.Provinces {
		for _, isp := range public.Isps {
			Conf.Province = province
			Conf.Isp = isp
			pcdn := getPcdn("dummy")
			parts := strings.Split(pcdn, ":")
			if len(parts) != 2 {
				return
			}
			pcdnIsp, pcdnArea, _ := localUtil.GetLocate(parts[0], IpParser)
			reqArea, _ := schedUtil.ProvinceAreaRelation(province)
			if reqArea != pcdnArea {
				sublogger.Error().Str("reqArea", reqArea).Str("pcdnArea", pcdnArea).Str("pcdn", pcdn).
					Str("reqIsp", isp).Str("pcdnIsp", pcdnIsp).Msg("area chk err")
				totalAreaNotMatch++
				areaNotCoverCntMap[reqArea+"_"+isp]++
			}
			if isp != pcdnIsp {
				sublogger.Error().Str("reqArea", reqArea).Str("pcdnArea", pcdnArea).Str("pcdn", pcdn).
					Str("reqIsp", isp).Str("pcdnIsp", pcdnIsp).Msg("isp chk err")
				totalIspNotMatch++
			}
			totalReqCnt++
		}
	}
	sublogger.Info().Int("totalAreaNotMatch", totalAreaNotMatch).Int("totalIspNotMatch", totalIspNotMatch).
		Int("areaNotMatch", len(areaNotCoverCntMap)).Int("totalReqCnt", totalReqCnt).Msg("Pcdns")
	for area, cnt := range areaNotCoverCntMap {
		sublogger.Info().Str("area", area).Int("cnt", cnt).Msg("area not match cnt")
	}
}

func LoopPcdn() {
	for i := 0; i < Conf.N; i++ {
		pcdn := getPcdn(fmt.Sprintf("%d", i))
		sublogger.Info().Str("pcdn", pcdn).Msg("")
	}
}

func Playcheck() {
	resp := playcheck(Conf.Ip + ":8080")
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

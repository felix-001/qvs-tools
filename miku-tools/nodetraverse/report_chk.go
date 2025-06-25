package nodetraverse

import (
	"log"
	"middle-source-analysis/callback"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
)

func RegisterReportChk() {
	Register(&ReportChk{nodeMap: make(map[string]*public.RtNode)})
}

type ReportChk struct {
	nodeMap map[string]*public.RtNode
}

func (m *ReportChk) OnIp(node *public.RtNode, ip *public.RtIpStatus) {
}

func (m *ReportChk) OnNode(node *public.RtNode, ipParser *ipdb.City, callback callback.Callback) {
	if !node.IsDynamic {
		return
	}
	report, err := callback.GetNodeAllStreams(node.Id)
	if err != nil || report == nil {
		log.Printf("[ReportChk][OnNode], nodeId:%s, err:%v\n", node.Id, err)
		return
	}
	for _, stream := range report.Streams {
		if stream.Key == "687423rCNRItVfMm_1024h_dy" {
			m.nodeMap[node.Id] = node
		}
	}
}

func (m *ReportChk) Done(ipParser *ipdb.City) {
	for nodeId, node := range m.nodeMap {
		area, isp := GetNodeAreaIsp(node, ipParser)
		log.Printf("[ReportChk][Done], nodeId:%s, %s, %s\n", nodeId, area, isp)
	}
}

func GetIpAreaIsp(ip string, ipParser *ipdb.City) (string, string) {
	locate, err := ipParser.Find(ip)
	if err != nil {
		log.Printf("查找IP %s 的位置信息失败: %v", ip, err)
		return "", ""
	}
	if locate.Isp == "" {
		log.Println(locate.Country, locate.Region, locate.City, ip)
	}
	area := monitorUtil.ProvinceAreaRelation(locate.Region)
	return area, locate.Isp
}

func GetNodeAreaIsp(node *public.RtNode, ipParser *ipdb.City) (string, string) {
	for _, ip := range node.Ips {
		if publicUtil.IsPrivateIP(ip.Ip) {
			continue
		}
		area, isp := GetIpAreaIsp(ip.Ip, ipParser)
		if area == "" || isp == "" {
			continue
		}
		return area, isp
	}
	return "", ""
}

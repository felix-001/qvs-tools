package nodemgr

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/miku/streammgr"
	"mikutool/public/util"

	schedModel "github.com/qbox/mikud-live/cmd/sched/model"
	commonUtil "github.com/qbox/mikud-live/common/util"
)

func GetPcdnFromSchedAPI(skipReport, skipRoot bool, conf *config.Config, nodeMgr *NodeMgr) (string, string) {
	addr := "http://10.34.146.62:6060/api/v1/nodes?level=default&dimension=area&mode=detail&ipversion=ipv4"
	resp, err := util.Get(addr)
	if err != nil {
		log.Println("get nodes err:", err)
		return "", ""
	}
	//fmt.Println(resp)
	areaNodesMap := make(map[string][]*schedModel.NodeIpsPair)
	if err := json.Unmarshal([]byte(resp), &areaNodesMap); err != nil {
		log.Println("unmarshal err:", err)
		return "", ""
	}
	key := fmt.Sprintf("area_isp_group_%s_%s", conf.Area, conf.Isp)
	nodes, ok := areaNodesMap[key]
	if !ok {
		log.Println("area isp not found nodes")
		return "", ""
	}
	if len(nodes) == 0 {
		log.Println("nodes len is 0")
		return "", ""
	}
	nodesMap := streammgr.GetNodesByStreamId(conf)
	streamNodes := nodesMap[key]
	if streamNodes == nil {
		log.Println("not found stream nodes")
	}
	pcdn := ""
	var selectNode *schedModel.NodeIpsPair
	for _, nodeInfo := range nodes {
		if skipReport {
			for _, detail := range streamNodes {
				if nodeInfo.Node.Id == detail.NodeId {
					log.Println("skip node:", nodeInfo.Node.Id)
					continue
				}
			}
		}
		if skipRoot {
			if nodeMgr.GetRootNodeByNodeId(nodeInfo.Node.Id) != nil {
				log.Println("skip root node:", nodeInfo.Node.Id)
				continue
			}
		}
		for _, ipInfo := range nodeInfo.Ips {
			if ipInfo.IsIPv6 {
				continue
			}
			if commonUtil.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			pcdn = fmt.Sprintf("%s:%d", ipInfo.Ip, nodeInfo.Node.StreamdPorts.Http)
			selectNode = nodeInfo
			break
		}
	}
	if pcdn == "" {
		log.Println("pcdn empty")
		return "", ""
	}
	log.Println("selected node:", selectNode.Node.Id, "pcdn:", pcdn)
	return selectNode.Node.Id, pcdn
}

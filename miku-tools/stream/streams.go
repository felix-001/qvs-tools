package stream

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"middle-source-analysis/config"
	"middle-source-analysis/node"
	"middle-source-analysis/public"
	localUtil "middle-source-analysis/util"
	"os/exec"
	"time"

	"github.com/qbox/mikud-live/common/model"
)

func getStreamSourceNodeMap(bkt string) (map[string][]string, map[string][]string) {
	streamSourceNodesMap := make(map[string][]string)
	notServingNodesMap := make(map[string][]string)
	for nodeId, streamInfo := range NodeStremasMap {
		for _, stream := range streamInfo.Streams {
			if stream.Bucket != bkt {
				continue
			}
			if stream.RelayBandwidth == 0 || stream.RelayType != 2 {
				continue
			}
			node, ok := node.AllNodesMap[nodeId]
			if !ok {
				log.Println("getStreamSourceNodeMap", node.Id, "not found in allNodesMap")
				continue
			}
			if node.RuntimeStatus != "Serving" {
				notServingNodesMap[nodeId] = append(notServingNodesMap[nodeId], nodeId)
				continue
			}
			streamSourceNodesMap[stream.Key] = append(streamSourceNodesMap[stream.Key], nodeId)
		}
	}
	return streamSourceNodesMap, notServingNodesMap
}

// var streamRatioHdr = "流ID, ISP, 在线人数, 回源节点个数, 拉流带宽, 回源带宽, 放大比, 边缘回源节点个数, 边缘回源节点详情, root回源节点个数, root回源节点详情, 静态回源节点个数, 静态回源节点详情, 离线回源节点个数, 离线回源节点详情, new \n"
var streamRatioHdr = "流ID, ISP, 在线人数, 回源节点个数, 静态回源节点个数, 拉流带宽, 回源带宽, 放大比\n"

func dumpStreamsDetail(bkt string) {
	streamSourceNodesMap, notServingNodesMap := getStreamSourceNodeMap(bkt)
	streamRatioCsv := streamRatioHdr
	for streamId, streamDetail := range StreamDetailMap {
		var streamTotalBw, streamTotalRelayBw float64
		var streamTotalOnlineNum, streamTotalEdgeNodeCount, streamTotalRootNodeCount int
		for _, detail := range streamDetail {
			for _, streamInfo := range detail {
				streamTotalBw += streamInfo.Bw
				streamTotalRelayBw += streamInfo.RelayBw
				streamTotalOnlineNum += int(streamInfo.OnlineNum)
				//streamTotalEdgeNodeCount += len(streamInfo.EdgeNodes)
				//streamTotalRootNodeCount += len(streamInfo.RootNodes)
			}
		}
		streamRatioCsv += fmt.Sprintf("%s, %d, %d, %d, %d, %d, %.1f, %.1f, %+v, %+v\n", streamId,
			streamTotalOnlineNum, streamTotalEdgeNodeCount, streamTotalRootNodeCount,
			len(streamSourceNodesMap[streamId]), len(notServingNodesMap[streamId]),
			streamTotalRelayBw, streamTotalBw/streamTotalRelayBw, streamSourceNodesMap[streamId],
			notServingNodesMap[streamId])
	}
	file := fmt.Sprintf("streams-%d.csv", time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(streamRatioCsv), 0644)
	if err != nil {
		log.Println(err)
	}
}

func getStreamInfo(nodeId, streamId string) *model.StreamInfoRT {
	streamInfo := NodeStremasMap[nodeId]
	for _, stream := range streamInfo.Streams {
		if stream.Key == streamId {
			return stream
		}
	}
	return nil
}

func GetStreamSchedInfos(conf *config.Config) []public.SchedInfo {
	streamSourceNodesMap, _ := GetStreamSourceNodeMap(conf.Bucket)
	nodeIds := streamSourceNodesMap[conf.Stream]
	if len(nodeIds) == 0 {
		log.Println("get stream source nodes err, len(nodeIds) = 0", conf.Bucket, conf.Stream)
		return nil
	}
	schedInfos := make([]public.SchedInfo, 0)
	for _, nodeId := range nodeIds {
		streamInfo := GetStreamInfo(nodeId, conf.Stream)
		node := node.AllNodesMap[nodeId]
		if node == nil {
			log.Println("get node err", nodeId)
			continue
		}
		startTime := GetStreamNodeInfo(streamInfo.Pusher[0].ConnectId, nodeId)
		/*
			log.Println("reqId:", streamInfo.Pusher[0].ConnectId, "startTime:", startTime, "nodeId:", nodeId,
				"machineId", node.MachineId)
		*/
		schedInfo := public.SchedInfo{
			ConnId:    streamInfo.Pusher[0].ConnectId,
			StartTime: startTime,
			NodeId:    nodeId,
			MachindId: node.MachineId,
		}
		schedInfos = append(schedInfos, schedInfo)
	}
	return schedInfos
}

const (
	NodeTypeRoot    = "root"
	NodeTypeEdge    = "edge"
	NodeTypeOffline = "offline"
	NodeTypeStatic  = "static"
)

func getNodeType(nodeInfo *model.RtNode) string {
	if !nodeInfo.IsDynamic {
		return NodeTypeStatic
	}
	if nodeInfo.IsDynamic && nodeInfo.RuntimeStatus != "Serving" {
		return NodeTypeOffline
	}
	if _, ok := node.AllRootNodesMapByNodeId[nodeInfo.Id]; ok {
		return NodeTypeRoot
	}
	return NodeTypeEdge
}

type StreamInfoDetail struct {
	RelayBw     float64
	Bw          float64
	OnlineNum   int
	OriginNodes map[string][]string // key: root/leaf
}

func dumpStreams() {
	// key1: streamId key2: isp key3: area
	streamDetailMap := make(map[string]map[string]map[string]*StreamInfoDetail)
	for _, node := range node.AllNodesMap {
		report := NodeStremasMap[node.Id]
		if report == nil {
			continue
		}
		if time.Now().Unix()-report.LastUpdateTime > 300 {
			continue
		}
		if node.NatType == "nat1" {
			continue
		}
		isp, area, _ := localUtil.GetNodeLocate(node, IpParser)
		if isp == "" || area == "" {
			continue
		}
		for _, streamInfoRT := range report.Streams {
			if Conf.Bucket != streamInfoRT.Bucket {
				continue
			}
			streamId := streamInfoRT.Key
			if _, ok := streamDetailMap[streamId]; !ok {
				streamDetailMap[streamId] = make(map[string]map[string]*StreamInfoDetail)
			}
			if _, ok := streamDetailMap[streamId][isp]; !ok {
				streamDetailMap[streamId][isp] = make(map[string]*StreamInfoDetail)
			}
			detail, ok := streamDetailMap[streamId][isp][area]
			if !ok {
				detail = &StreamInfoDetail{
					OriginNodes: make(map[string][]string),
				}
				streamDetailMap[streamId][isp][area] = detail
			}
			onlineNum, bw := localUtil.GetStreamDetail(streamInfoRT)
			detail.OnlineNum += onlineNum
			detail.Bw += bw
			if streamInfoRT.RelayType != 2 {
				continue
			}
			detail.RelayBw += localUtil.Bps2Mbps(streamInfoRT.RelayBandwidth)
			nodeType := getNodeType(node)
			if _, ok := detail.OriginNodes[nodeType]; !ok {
				detail.OriginNodes[nodeType] = make([]string, 0)
			}
			detail.OriginNodes[nodeType] = append(detail.OriginNodes[nodeType], node.Id)
		}
	}
	//saveStreamDetail(streamDetailMap)
	dumpCsv(streamDetailMap)
}

func dumpCsv(streamDetailMap map[string]map[string]map[string]*StreamInfoDetail) {
	areaCsv := "流id, isp, 大区, 在线人数, 拉流带宽, 回源带宽, leaf回源节点数, root回源节点数, 放大比\n"
	ispCsv := "流id, isp, 在线人数, 拉流带宽, 回源带宽, leaf回源节点数, root回源节点数, 放大比\n"
	staticNodeCsv := "流id, 在线人数, 静态回源节点数\n"
	for streamId, ispMap := range streamDetailMap {
		totalOnlineNum := 0
		totalStaticNodeCnt := 0
		for isp, areaMap := range ispMap {
			ispTotal := StreamInfoDetail{
				OriginNodes: make(map[string][]string),
			}
			for area, detail := range areaMap {
				leafCnt := len(detail.OriginNodes[NodeTypeEdge])
				rootCnt := len(detail.OriginNodes[NodeTypeRoot])
				areaCsv += fmt.Sprintf("%s, %s, %s, %d, %.1f, %.1f, %d, %d, %.1f\n",
					streamId, isp, area, detail.OnlineNum, detail.Bw, detail.RelayBw,
					leafCnt, rootCnt, detail.Bw/detail.RelayBw)

				ispTotal.Bw += detail.Bw
				ispTotal.OnlineNum += detail.OnlineNum
				ispTotal.RelayBw += detail.RelayBw
				for nodeType, nodes := range detail.OriginNodes {
					ispTotal.OriginNodes[nodeType] = append(ispTotal.OriginNodes[nodeType],
						nodes...)
				}
				totalStaticNodeCnt += len(detail.OriginNodes[NodeTypeStatic])
				totalOnlineNum += detail.OnlineNum
			}
			leafCnt := len(ispTotal.OriginNodes[NodeTypeEdge])
			rootCnt := len(ispTotal.OriginNodes[NodeTypeRoot])
			ispCsv += fmt.Sprintf("%s, %s, %d, %.1f, %.1f, %d, %d, %.1f\n",
				streamId, isp, ispTotal.OnlineNum, ispTotal.Bw, ispTotal.RelayBw,
				leafCnt, rootCnt, ispTotal.Bw/ispTotal.RelayBw)
		}
		staticNodeCsv += fmt.Sprintf("%s, %d, %d\n", streamId, totalOnlineNum, totalStaticNodeCnt)
	}
	saveFile(fmt.Sprintf("streams-area-%d.csv", time.Now().Unix()), areaCsv)
	saveFile(fmt.Sprintf("streams-isp-%d.csv", time.Now().Unix()), ispCsv)
	saveFile(fmt.Sprintf("streams-static-%d.csv", time.Now().Unix()), staticNodeCsv)
}

/*
func  saveStreamDetail(streamDetailMap map[StreamKey]*StreamInfoDetail) {
	bytes, err := json.Marshal(streamDetailMap)
	if err != nil {
		log.Println(err)
		return
	}
	saveFile(fmt.Sprintf("streams-%d.json", time.Now().Unix()), string(bytes))
}
*/

func getStreamClientCnt(streamInfoRT *model.StreamInfoRT) int {
	cnt := 0
	for _, player := range streamInfoRT.Players {
		cnt += len(player.Ips)
	}
	return cnt
}

func getStreamOnlineNum(streamInfoRT *model.StreamInfoRT) int {
	onlineNum := 0
	for _, player := range streamInfoRT.Players {
		for _, ipInfo := range player.Ips {
			onlineNum += int(ipInfo.OnlineNum)
		}
	}
	return onlineNum
}

func getStreamBw(streamInfoRT *model.StreamInfoRT) float64 {
	var bw float64
	for _, player := range streamInfoRT.Players {
		for _, ipInfo := range player.Ips {
			bw += float64(ipInfo.Bandwidth * 8 / 1e6)
		}
	}
	return bw
}

func dumpStreamDetail(detail *public.StreamInfo, lastStreamId, streamId, lastIsp, isp string) string {
	csv := ""
	for nodeType, streamDetail := range detail.NodeStreamMap {
		lastNodeType := ""
		for nodeId, streamInfoRT := range streamDetail {
			sid := ""
			if lastStreamId != streamId {
				sid = streamId
			}
			tmpIsp := ""
			if lastIsp != isp {
				tmpIsp = isp
			}
			tmpNodeType := ""
			if lastNodeType != nodeType {
				tmpNodeType = nodeType
			}
			lastNodeType = nodeType
			lastStreamId = streamId
			lastIsp = isp
			node := node.AllNodesMap[nodeId]
			_, area, province := localUtil.GetNodeLocate(node, IpParser)
			csv += fmt.Sprintf("%s, %s, %s, %s, %d, %.1f, %d, %s, %s, %s\n",
				sid, tmpIsp, tmpNodeType, nodeId, getStreamClientCnt(streamInfoRT),
				getStreamBw(streamInfoRT), getStreamOnlineNum(streamInfoRT),
				streamInfoRT.Pusher[0].ConnectId, area, province)
		}
	}
	return csv
}

func saveFile(filename, csv string) {
	file := fmt.Sprintf(filename)
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
	cmd := exec.Command("qup", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

var streamDetailHdr = "流ID, ISP, 节点类型, 节点ID, 客户端个数, 带宽, 在线人数, connId, 大区, 省份\n"
var streamStaticNodeCntHdr = "流ID, 在线人数, 静态节点个数, 拉流带宽, 回源带宽, 放大比, 节点详情\n"

func saveStreamsInfoToCSV() {
	csv := streamRatioHdr
	streamDetailCsv := streamDetailHdr
	streamStaticNodeCntCsv := streamStaticNodeCntHdr
	lastStreamId := ""
	for streamId, ispDetail := range streamInfoMap {
		lastIsp := ""
		totalOnlineNum := 0
		totalStaticNodeCnt := 0
		var totalBw, totalRelayBw float64
		var staticNodes []string
		for isp, detail := range ispDetail {
			ratio := detail.Bw / detail.RelayBw
			nodeCnt := len(detail.NodeStreamMap[NodeTypeEdge]) +
				len(detail.NodeStreamMap[NodeTypeRoot]) +
				len(detail.NodeStreamMap[NodeTypeStatic])
			csv += fmt.Sprintf("%s, %s, %d, %d, %.1f, %.1f, %.1f\n",
				streamId, isp, detail.OnlineNum, nodeCnt,
				detail.Bw, detail.RelayBw, ratio)
			streamDetailCsv += dumpStreamDetail(detail, lastStreamId,
				streamId, lastIsp, isp)

			totalOnlineNum += int(detail.OnlineNum)
			totalStaticNodeCnt += len(detail.NodeStreamMap[NodeTypeStatic])
			totalBw += detail.Bw
			totalRelayBw += detail.RelayBw
			for nodeId, _ := range detail.NodeStreamMap[NodeTypeStatic] {
				staticNodes = append(staticNodes, nodeId)
			}
		}
		streamStaticNodeCntCsv += fmt.Sprintf("%s, %d, %d, %.1f, %.1f, %.1f, %+v\n",
			streamId, totalOnlineNum, totalStaticNodeCnt, totalBw,
			totalRelayBw, totalBw/totalRelayBw, staticNodes)
	}

	saveFile(fmt.Sprintf("streams-%d.csv", time.Now().Unix()), csv)
	saveFile(fmt.Sprintf("streamsDetail-%d.csv", time.Now().Unix()), streamDetailCsv)
	saveFile(fmt.Sprintf("streams-static-cnt-%d.csv", time.Now().Unix()), streamStaticNodeCntCsv)
}

func Report() {
	data, err := json.Marshal(NodeStremasMap)
	if err != nil {
		log.Println(err)
		return
	}
	saveFile(fmt.Sprintf("report-%d.json", time.Now().Unix()), string(data))
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/qbox/mikud-live/common/model"
)

func (s *Parser) getStreamSourceNodeMap(bkt string) (map[string][]string, map[string][]string) {
	streamSourceNodesMap := make(map[string][]string)
	notServingNodesMap := make(map[string][]string)
	for nodeId, streamInfo := range s.nodeStremasMap {
		for _, stream := range streamInfo.Streams {
			if stream.Bucket != bkt {
				continue
			}
			if stream.RelayBandwidth == 0 || stream.RelayType != 2 {
				continue
			}
			node, ok := s.allNodesMap[nodeId]
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

func (s *Parser) dumpStreamsDetail(bkt string) {
	streamSourceNodesMap, notServingNodesMap := s.getStreamSourceNodeMap(bkt)
	streamRatioCsv := streamRatioHdr
	for streamId, streamDetail := range s.streamDetailMap {
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

func (s *Parser) getStreamInfo(nodeId, streamId string) *model.StreamInfoRT {
	streamInfo := s.nodeStremasMap[nodeId]
	for _, stream := range streamInfo.Streams {
		if stream.Key == streamId {
			return stream
		}
	}
	return nil
}

type SchedInfo struct {
	ConnId    string
	StartTime int64
	NodeId    string
	MachindId string
}

func (s *Parser) getStreamSchedInfos() []SchedInfo {
	streamSourceNodesMap, _ := s.getStreamSourceNodeMap(s.conf.Bucket)
	nodeIds := streamSourceNodesMap[s.conf.Stream]
	if len(nodeIds) == 0 {
		log.Println("get stream source nodes err, len(nodeIds) = 0", s.conf.Bucket, s.conf.Stream)
		return nil
	}
	schedInfos := make([]SchedInfo, 0)
	for _, nodeId := range nodeIds {
		streamInfo := s.getStreamInfo(nodeId, s.conf.Stream)
		node := s.allNodesMap[nodeId]
		if node == nil {
			log.Println("get node err", nodeId)
			continue
		}
		startTime := s.GetStreamNodeInfo(streamInfo.Pusher[0].ConnectId, nodeId)
		/*
			log.Println("reqId:", streamInfo.Pusher[0].ConnectId, "startTime:", startTime, "nodeId:", nodeId,
				"machineId", node.MachineId)
		*/
		schedInfo := SchedInfo{
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

func (s *Parser) getNodeType(node *model.RtNode) string {
	if !node.IsDynamic {
		return NodeTypeStatic
	}
	if node.IsDynamic && node.RuntimeStatus != "Serving" {
		return NodeTypeOffline
	}
	if _, ok := s.allRootNodesMapByNodeId[node.Id]; ok {
		return NodeTypeRoot
	}
	return NodeTypeEdge
}

func (s *Parser) getNodeIsp(node *model.RtNode) string {
	for _, ipInfo := range node.Ips {
		if !checkIp(ipInfo) {
			continue
		}
		return ipInfo.Isp
	}
	return "unknow"
}

func (s *Parser) dumpStreams_bak() {
	streamInfoMap := make(map[string]map[string]*StreamInfo) // key1: streamId key2: isp
	for _, node := range s.allNodesMap {
		report := s.nodeStremasMap[node.Id]
		if report == nil {
			continue
		}
		if time.Now().Unix()-report.LastUpdateTime > 300 {
			continue
		}
		isp := s.getNodeIsp(node)
		for _, streamInfoRT := range report.Streams {
			if s.conf.Bucket != streamInfoRT.Bucket {
				continue
			}
			_, ok := streamInfoMap[streamInfoRT.Key]
			if !ok {
				streamInfoMap[streamInfoRT.Key] = make(map[string]*StreamInfo)
			}
			streamInfo, ok := streamInfoMap[streamInfoRT.Key][isp]
			if !ok {
				streamInfo = &StreamInfo{}
				streamInfo.NodeStreamMap = make(map[string]map[string]*model.StreamInfoRT)
				streamInfoMap[streamInfoRT.Key][isp] = streamInfo
			}
			onlineNum, bw := s.getStreamDetail(streamInfoRT)
			streamInfo.Bw += bw
			streamInfo.RelayBw += convertMbps(streamInfoRT.RelayBandwidth)
			streamInfo.OnlineNum += uint32(onlineNum)
			if /*streamInfoRT.RelayBandwidth == 0 || */ streamInfoRT.RelayType != 2 {
				continue
			}
			nodeType := s.getNodeType(node)
			if _, ok := streamInfo.NodeStreamMap[nodeType]; !ok {
				streamInfo.NodeStreamMap[nodeType] = make(map[string]*model.StreamInfoRT)
			}
			streamInfo.NodeStreamMap[nodeType][node.Id] = streamInfoRT
		}
	}
	s.streamInfoMap = streamInfoMap
	log.Println("streams:", len(streamInfoMap))
	s.saveStreamsInfoToCSV()
}

type StreamKey struct {
	StreamId string
	Isp      string
	Area     string
}

type StreamInfoDetail struct {
	RelayBw     float64
	Bw          float64
	OnlineNum   int
	OriginNodes map[string][]string // key: root/leaf
}

func (s *Parser) dumpStreams() {
	streamDetailMap := make(map[StreamKey]*StreamInfoDetail)
	for _, node := range s.allNodesMap {
		report := s.nodeStremasMap[node.Id]
		if report == nil {
			continue
		}
		if time.Now().Unix()-report.LastUpdateTime > 300 {
			continue
		}
		if node.NatType == "nat1" {
			continue
		}
		isp, area, _ := getNodeLocate(node, s.ipParser)
		if isp == "" || area == "" {
			s.logger.Warn().Str("node", node.Id).Msg("get node locate err")
			continue
		}
		for _, streamInfoRT := range report.Streams {
			if s.conf.Bucket != streamInfoRT.Bucket {
				continue
			}
			key := StreamKey{
				StreamId: streamInfoRT.Key,
				Area:     area,
				Isp:      isp,
			}
			streamDetail, ok := streamDetailMap[key]
			if !ok {
				streamDetail = &StreamInfoDetail{
					OriginNodes: make(map[string][]string),
				}
				streamDetailMap[key] = streamDetail
			}
			onlineNum, bw := s.getStreamDetail(streamInfoRT)
			streamDetail.OnlineNum += onlineNum
			streamDetail.Bw += bw
			if streamInfoRT.RelayType != 2 {
				continue
			}
			streamDetail.RelayBw += Bps2Mbps(streamInfoRT.RelayBandwidth)
			nodeType := s.getNodeType(node)
			if _, ok := streamDetail.OriginNodes[nodeType]; !ok {
				streamDetail.OriginNodes[nodeType] = make([]string, 0)
			}
			streamDetail.OriginNodes[nodeType] = append(streamDetail.OriginNodes[nodeType], node.Id)
		}
	}
	s.saveStreamDetail(streamDetailMap)
}

func (s StreamKey) MarshalText() (text []byte, err error) {
	return []byte(fmt.Sprintf("%s_%s_%s", s.StreamId, s.Area, s.Isp)), nil
}

func (s *Parser) saveStreamDetail(streamDetailMap map[StreamKey]*StreamInfoDetail) {
	bytes, err := json.Marshal(streamDetailMap)
	if err != nil {
		log.Println(err)
		return
	}
	s.saveFile(fmt.Sprintf("streams-%d.json", time.Now().Unix()), string(bytes))
}

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

func (s *Parser) dumpStreamDetail(detail *StreamInfo, lastStreamId, streamId, lastIsp, isp string) string {
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
			node := s.allNodesMap[nodeId]
			_, area, province := getNodeLocate(node, s.ipParser)
			csv += fmt.Sprintf("%s, %s, %s, %s, %d, %.1f, %d, %s, %s, %s\n",
				sid, tmpIsp, tmpNodeType, nodeId, getStreamClientCnt(streamInfoRT),
				getStreamBw(streamInfoRT), getStreamOnlineNum(streamInfoRT),
				streamInfoRT.Pusher[0].ConnectId, area, province)
		}
	}
	return csv
}

func (s *Parser) saveFile(filename, csv string) {
	file := fmt.Sprintf(filename)
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
	cmd := exec.Command("./qup", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

var streamDetailHdr = "流ID, ISP, 节点类型, 节点ID, 客户端个数, 带宽, 在线人数, connId, 大区, 省份\n"
var streamStaticNodeCntHdr = "流ID, 在线人数, 静态节点个数, 拉流带宽, 回源带宽, 放大比, 节点详情\n"

func (s *Parser) saveStreamsInfoToCSV() {
	csv := streamRatioHdr
	streamDetailCsv := streamDetailHdr
	streamStaticNodeCntCsv := streamStaticNodeCntHdr
	lastStreamId := ""
	for streamId, ispDetail := range s.streamInfoMap {
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
			streamDetailCsv += s.dumpStreamDetail(detail, lastStreamId,
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

	s.saveFile(fmt.Sprintf("streams-%d.csv", time.Now().Unix()), csv)
	s.saveFile(fmt.Sprintf("streamsDetail-%d.csv", time.Now().Unix()), streamDetailCsv)
	s.saveFile(fmt.Sprintf("streams-static-cnt-%d.csv", time.Now().Unix()), streamStaticNodeCntCsv)
}

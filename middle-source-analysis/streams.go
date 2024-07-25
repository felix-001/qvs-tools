package main

import (
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

var streamRatioHdr = "流ID, ISP, 在线人数, 回源节点个数, 拉流带宽, 回源带宽, 放大比, 回源节点详情\n"

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
				streamTotalEdgeNodeCount += len(streamInfo.EdgeNodes)
				streamTotalRootNodeCount += len(streamInfo.RootNodes)
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
		log.Println("get stream source nodes err", s.conf.Bucket, s.conf.Stream)
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
		log.Println("reqId:", streamInfo.Pusher[0].ConnectId, "startTime:", startTime, "nodeId:", nodeId,
			"machineId", node.MachineId)
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
)

func (s *Parser) getNodeType(node *model.RtNode) string {
	if node.RuntimeStatus != "Serving" {
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

func (s *Parser) dumpStreams() {
	streamInfoMap := make(map[string]map[string]*StreamInfo) // key1: streamId key2: isp
	for _, node := range s.allNodesMap {
		report := s.nodeStremasMap[node.Id]
		if report == nil {
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
				streamInfoMap[streamInfoRT.Key][isp] = streamInfo
			}
			onlineNum, bw := s.getStreamDetail(streamInfoRT)
			streamInfo.Bw += bw
			streamInfo.RelayBw += convertMbps(streamInfoRT.RelayBandwidth)
			streamInfo.OnlineNum += uint32(onlineNum)
			if streamInfoRT.RelayBandwidth == 0 || streamInfoRT.RelayType != 2 {
				continue
			}
			nodeType := s.getNodeType(node)
			switch nodeType {
			case NodeTypeRoot:
				streamInfo.RootNodes = append(streamInfo.RootNodes, node.Id)
			case NodeTypeOffline:
				streamInfo.OfflineNodes = append(streamInfo.OfflineNodes, node.Id)
			default:
				streamInfo.EdgeNodes = append(streamInfo.EdgeNodes, node.Id)
			}
		}
	}
	s.streamInfoMap = streamInfoMap
	log.Println("streams:", len(streamInfoMap))
	s.saveStreamsInfoToCSV()
}

func (s *Parser) saveStreamsInfoToCSV() {
	csv := streamRatioHdr
	for streamId, ispDetail := range s.streamInfoMap {
		for isp, detail := range ispDetail {
			ratio := detail.Bw / detail.RelayBw
			nodeCnt := len(detail.EdgeNodes) + len(detail.RootNodes)
			nodesDetail := fmt.Sprintf("edgeNodesCnt: %d edgeNodesDetail: %+v rootNodesCnt: %d rootNodesDetail: %+v offlineNodesCnt: %d offlineNodesDetail: %+v",
				len(detail.EdgeNodes), detail.EdgeNodes, len(detail.RootNodes), detail.RootNodes,
				len(detail.OfflineNodes), detail.OfflineNodes)
			csv += fmt.Sprintf("%s, %s, %d, %d, %.1f, %.1f, %.1f, %s\n", streamId, isp, detail.OnlineNum, nodeCnt,
				detail.Bw, detail.RelayBw, ratio, nodesDetail)
		}
	}

	file := fmt.Sprintf("streams-%d.csv", time.Now().Unix())
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

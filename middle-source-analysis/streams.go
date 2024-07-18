package main

import (
	"fmt"
	"io/ioutil"
	"log"
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

var streamRatioHdr = "流ID, 在线人数, 边缘节点个数, ROOT节点个数, 回源节点个数, 非serving的回源节点个数, 回源带宽, 放大比, 回源节点详情, 非serving回源节点详情\n"

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

func (s *Parser) dumpStreamDetail(bucket, stream string) {
	streamSourceNodesMap, _ := s.getStreamSourceNodeMap(bucket)
	nodeIds := streamSourceNodesMap[stream]
	if len(nodeIds) == 0 {
		log.Println("get stream source nodes err", bucket, stream)
		return
	}
	for _, nodeId := range nodeIds {
		streamInfo := s.getStreamInfo(nodeId, stream)
		node := s.allNodesMap[nodeId]
		if node == nil {
			log.Println("get node err", nodeId)
			continue
		}
		startTime := s.GetStreamNodeInfo(streamInfo.Pusher[0].ConnectId, nodeId)
		log.Println("reqId:", streamInfo.Pusher[0].ConnectId, "startTime:", startTime, "nodeId:", nodeId, "machineId", node.MachineId)

	}
}

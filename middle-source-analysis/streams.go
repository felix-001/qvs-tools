package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

func (s *Parser) getStreamSourceNodeMap(bkt string) map[string][]string {
	streamSourceNodesMap := make(map[string][]string)
	for node, streamInfo := range s.nodeStremasMap {
		for _, stream := range streamInfo.Streams {
			if stream.Bucket != bkt {
				continue
			}
			if stream.RelayBandwidth == 0 || stream.RelayType != 2 {
				continue
			}
			streamSourceNodesMap[stream.Key] = append(streamSourceNodesMap[stream.Key], node)
		}
	}
	return streamSourceNodesMap
}

var streamRatioHdr = "流ID, 在线人数, 边缘节点个数, ROOT节点个数, 回源节点个数, 回源带宽, 放大比, 回源节点详情\n"

func (s *Parser) dumpStreamsDetail(bkt string) {
	streamSourceNodesMap := s.getStreamSourceNodeMap(bkt)
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
		streamRatioCsv += fmt.Sprintf("%s, %d, %d, %d, %d, %.1f, %.1f, %+v\n", streamId,
			streamTotalOnlineNum, streamTotalEdgeNodeCount,
			streamTotalRootNodeCount, len(streamSourceNodesMap[streamId]),
			streamTotalRelayBw, streamTotalBw/streamTotalRelayBw, streamSourceNodesMap[streamId])
	}
	file := fmt.Sprintf("streams-%d.csv", time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(streamRatioCsv), 0644)
	if err != nil {
		log.Println(err)
	}
}

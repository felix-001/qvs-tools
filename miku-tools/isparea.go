package main

import (
	"log"
	"strings"

	schedUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
)

/*
流id - ISP - 大区
map[string]map[string]map[string]StreamInfo
*/
func (s *Parser) buildBucketStreamsInfo(bkt string) {
	s.streamDetailMap = make(map[string]map[string]map[string]*StreamInfo)
	for _, node := range s.allNodesMap {
		streamInfo := s.nodeStremasMap[node.Id]
		if !s.check(node, streamInfo) {
			continue
		}
		lastStream := ""
		isp, area, _ := getNodeLocate(node, s.ipParser)
		if isp == "" || area == "" {
			//log.Println("node", node.Id, "get ip locate err")
			continue
		}
		for _, stream := range streamInfo.Streams {
			if stream.Bucket != bkt {
				continue
			}
			if lastStream != "" && stream.Key == lastStream {
				log.Println("two samle stream in one node", "nodeid:", node.Id, "streamid", stream.Key,
					"relayBandwidth:", stream.RelayBandwidth)
			}
			if lastStream == "" {
				lastStream = stream.Key
			}
			/*
				if stream.RelayType != 0 {
					log.Println("stream", stream.Key, "node", node.Id, "relaytype:", stream.RelayType)
				}
			*/
			if _, ok := s.streamDetailMap[stream.Key]; !ok {
				s.streamDetailMap[stream.Key] = make(map[string]map[string]*StreamInfo)
			}
			if _, ok := s.streamDetailMap[stream.Key][isp]; !ok {
				s.streamDetailMap[stream.Key][isp] = make(map[string]*StreamInfo)
			}
			onlineNum, bw := s.getStreamDetail(stream)
			//isRoot := s.isRoot(node)
			if streamInfo, ok := s.streamDetailMap[stream.Key][isp][area]; !ok {
				s.streamDetailMap[stream.Key][isp][area] = &StreamInfo{
					OnlineNum: uint32(onlineNum),
					Bw:        bw,
					//RelayBw:   convertMbps(stream.RelayBandwidth),
				}
				streamInfo := s.streamDetailMap[stream.Key][isp][area]
				/*
					if !isRoot {
						streamInfo.EdgeNodes = append(streamInfo.EdgeNodes, node.Id)
					} else {
						streamInfo.RootNodes = append(streamInfo.RootNodes, node.Id)
					}
				*/
				if stream.RelayType == 2 {
					streamInfo.RelayBw = convertMbps(stream.RelayBandwidth)
				}
			} else {
				streamInfo.OnlineNum += uint32(onlineNum)
				streamInfo.Bw += bw
				if stream.RelayType == 2 {
					streamInfo.RelayBw += convertMbps(stream.RelayBandwidth)
				}
				/*
					if !isRoot {
						streamInfo.EdgeNodes = append(streamInfo.EdgeNodes, node.Id)
					} else {
						streamInfo.RootNodes = append(streamInfo.RootNodes, node.Id)
					}
				*/
			}
		}
	}
	log.Println("total:", len(s.streamDetailMap))
}

func (s *Parser) Province2Area() {
	parts := strings.Split(s.conf.Province, ",")

	result := ""
	for _, province := range parts {
		area, _ := schedUtil.ProvinceAreaRelation(province)
		result += area + ","
	}
	log.Println(result)
}

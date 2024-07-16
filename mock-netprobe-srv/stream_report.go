package main

import (
	"encoding/json"
	"fmt"
)

// bucket - stream - node

func (s *NetprobeSrv) StreamReport(paramMap map[string]string) string {
	body := paramMap["body"]
	stream := paramMap["stream"]
	bucket := paramMap["bucket"]

	if _, ok := s.streamReportMap[bucket]; !ok {
		s.streamReportMap[bucket] = make(map[string]map[string]map[string]int)
	}
	if _, ok := s.streamReportMap[bucket][stream]; !ok {
		s.streamReportMap[bucket][stream] = make(map[string]map[string]int)
	}

	nodeStreamInfoMap := make(map[string]map[string]int)

	//ipOnlineNumMap := map[string]int{}
	if err := json.Unmarshal([]byte(body), &nodeStreamInfoMap); err != nil {
		return fmt.Sprintf("unmashal err, %v", err)
	}
	for nodeId, ips := range nodeStreamInfoMap {
		s.streamReportMap[bucket][stream][nodeId] = ips
	}

	/*
		for bucket, bucketMap := range s.streamReportMap {
			for stream, streamMap := range bucketMap {
				for nodeId, nodeMap := range streamMap {
					for ip, onlineNum := range nodeMap {
						log.Println(bucket, stream, nodeId, ip, onlineNum)
					}
				}
			}
		}
	*/

	/*
		var ips []*model.IpInfo
		for ip, onlineNum := range ipOnlineNumMap {
			ipInfo := &model.IpInfo{
				Ip:        ip,
				OnlineNum: uint32(onlineNum),
			}
			ips = append(ips, ipInfo)
		}
		nodeStreamInfo := model.NodeStreamInfo{
			NodeId:         node,
			LastUpdateTime: time.Now().Unix(),
			Streams: []*model.StreamInfoRT{
				{
					AppName:    bucket,
					Bucket:     bucket,
					Key:        stream,
					StreamName: stream,
					Players: []*model.PlayerInfo{
						{
							Ips: ips,
						},
					},
				},
			},
		}

		bytes, err := json.Marshal(&nodeStreamInfo)
		if err != nil {
			return fmt.Sprintf("marshal err, %v", err)
		}

		_, err = s.redisCli.Set(context.Background(), "stream_report_"+node, bytes, time.Hour*24*30).Result()
		if err != nil {
			log.Println(err)
			return fmt.Sprintf("redis err, %v", err)
		}
	*/

	return "success"
}

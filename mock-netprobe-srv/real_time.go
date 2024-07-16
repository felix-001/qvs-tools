package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/qbox/mikud-live/common/model"
)

func (s *NetprobeSrv) Run() {
	for range time.Tick(time.Duration(10) * time.Second) {
		log.Println("update nodes, count:", len(s.nodes))
		for _, node := range s.nodes {
			for i := range node.Ips {
				node.Ips[i].IPStreamProbe.LowThresholdTime = time.Now().Unix() - 3*3600
				if extra, ok := s.nodeExtras[node.Id]; ok {
					log.Println("found extra", node.Id, extra.LowThresholdTime)
					node.Ips[i].IPStreamProbe.LowThresholdTime = extra.LowThresholdTime
				}
			}
			bytes, err := json.Marshal(node)
			if err != nil {
				log.Println(err)
				return
			}
			_, err = s.redisCli.HSet(context.Background(), model.NetprobeRtNodesMap, node.Id, bytes).Result()
			if err != nil {
				log.Printf("write node info to redis err, %+v\n", err)
			}
			/*
				nodeStreamInfo := model.NodeStreamInfo{
					Streams: []*model.StreamInfoRT{
						{
							StreamName: "test",
						},
					},
					NodeId:         node.Id,
					LastUpdateTime: time.Now().Unix(),
				}
				data, err := json.Marshal(nodeStreamInfo)
				if err != nil {
					log.Println(err)
					continue
				}
				_, err = s.redisCli.Set(context.Background(), "stream_report_"+node.Id, string(data), time.Hour*24*30).Result()
				if err != nil {
					log.Println(err)
				}
			*/
		}
	}
}

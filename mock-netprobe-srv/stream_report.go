package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/qbox/mikud-live/common/model"
)

// bucket - stream - node

func (s *NetprobeSrv) StreamReport(paramMap map[string]string) string {
	node := paramMap["node"]
	body := paramMap["body"]
	stream := paramMap["stream"]
	bucket := paramMap["bucket"]

	ipOnlineNumMap := map[string]int{}
	if err := json.Unmarshal([]byte(body), &ipOnlineNumMap); err != nil {
		return fmt.Sprintf("unmashal err, %v", err)
	}

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

	return "success"
}

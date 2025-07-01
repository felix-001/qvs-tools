package main

import (
	"fmt"
	"log"
	"middle-source-analysis/util"
)

func (s *Parser) stopStream() {
	var streams []string
	for _, node := range s.allNodesMap {
		if !node.IsDynamic {
			continue
		}
		report := s.nodeStremasMap[node.Id]
		if report == nil {
			continue
		}
		for _, streamInfoRT := range report.Streams {
			if s.conf.Bucket != streamInfoRT.Bucket {
				continue
			}
			streams = append(streams, streamInfoRT.Key)
		}
	}
	for _, stream := range streams {
		addr := fmt.Sprintf("http://dycold.mls.cn-east-1.qiniumiku.com/%s?stop", stream)
		_, err := util.MikuHttpReq("POST", addr, "", s.conf.Ak, s.conf.Sk)
		if err != nil {
			log.Println(err)
		}
	}
}

package stream

import (
	"fmt"
	"log"
	"middle-source-analysis/node"
	"middle-source-analysis/util"
)

func stopStream() {
	var streams []string
	for _, node := range node.AllNodesMap {
		if !node.IsDynamic {
			continue
		}
		report := NodeStremasMap[node.Id]
		if report == nil {
			continue
		}
		for _, streamInfoRT := range report.Streams {
			if Conf.Bucket != streamInfoRT.Bucket {
				continue
			}
			streams = append(streams, streamInfoRT.Key)
		}
	}
	for _, stream := range streams {
		addr := fmt.Sprintf("http://dycold.mls.cn-east-1.qiniumiku.com/%s?stop", stream)
		_, err := util.MikuHttpReq("POST", addr, "", Conf.Ak, Conf.Sk)
		if err != nil {
			log.Println(err)
		}
	}
}

package nodetraverse

import (
	"log"
	"middle-source-analysis/callback"

	public "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
)

func RegisterReportChk() {
	Register(&ReportChk{})
}

type ReportChk struct {
}

func (m *ReportChk) OnIp(node *public.RtNode, ip *public.RtIpStatus) {
}

func (m *ReportChk) OnNode(node *public.RtNode, ipParser *ipdb.City, callback callback.Callback) {
	if !node.IsDynamic {
		return
	}
	report, err := callback.GetNodeAllStreams(node.Id)
	if err != nil || report == nil {
		log.Printf("[ReportChk][OnNode], nodeId:%s, err:%v\n", node.Id, err)
		return
	}
	streamMap := make(map[string][]*public.StreamInfoRT)
	for _, stream := range report.Streams {
		streamMap[stream.Key] = append(streamMap[stream.Key], stream)
	}
	for key, streams := range streamMap {
		if len(streams) > 3 {
			log.Println("contain repeat stream", "key:", key, "node:", node.Id, "bucket:", streams[0].Bucket,
				"streams:", len(streams))
			for i, stream := range streams {
				for _, player := range stream.Players {
					for _, ipInfo := range player.Ips {
						log.Println("i:", i, "bucket:", stream.Bucket, "domain:", stream.Domain, "relayType:", stream.RelayType,
							"relayBandwidth:", stream.RelayBandwidth, "key:", key, "node:", node.Id, "ip:", ipInfo.Ip,
							"protocol:", player.Protocol, "bandwidth:", ipInfo.Bandwidth)
					}
				}
			}
		}
	}
}

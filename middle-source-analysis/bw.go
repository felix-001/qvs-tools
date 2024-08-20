package main

import (
	"fmt"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	zlog "github.com/rs/zerolog/log"
)

func (s *Parser) CalcTotalBw() {
	var totalBwMbps float64 = 0
	logger := zlog.Logger
	logger = logger.Level(3)
	for _, node := range s.allNodesMap {
		if !node.IsDynamic {
			continue
		}
		if !util.CheckNodeUsable(logger, node, consts.TypeLive) {
			//log.Printf("checkNode nodeId:%s, machineId:%s check not pass, type: %s\n", node.Id, node.MachineId, node.ResourceType)
			s.NodeUnavailableCnt++
			continue
		}
		if !checkDynamicNodesPort(node) {
			s.NodeNoPortsCnt++
			continue
		}
		for _, ipInfo := range node.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				s.PrivateIpCnt++
				continue
			}
			if ipInfo.IsIPv6 {
				s.IpV6Cnt++
				continue
			}
			if ipInfo.IPStreamProbe.State != model.StreamProbeStateSuccess {
				s.NetProbeStateErrIpCnt++
				continue
			}
			if ipInfo.IPStreamProbe.Speed < 8 && ipInfo.IPStreamProbe.MinSpeed < 6 {
				s.NetProbeSpeedErrIpCnt++
				continue
			}
			totalBwMbps += ((ipInfo.MaxOutMBps * 8) / 1000)
		}
	}
	fmt.Printf("totalBw: %.0f, NodeUnavailableCnt: %d, NodeNoPortsCnt: %d, PrivateIpCnt: %d, NetProbeStateErrIpCnt: %d, NetProbeStateErrIpCnt: %d, IpV6Cnt: %d\n",
		totalBwMbps, s.NodeUnavailableCnt, s.NodeNoPortsCnt, s.PrivateIpCnt, s.NetProbeStateErrIpCnt, s.NetProbeSpeedErrIpCnt, s.IpV6Cnt)
}

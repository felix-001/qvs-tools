package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	zlog "github.com/rs/zerolog/log"
)

func (s *Parser) CalcTotalBw() {

	var totalMaxBwGbps float64 = 0
	var rawTotalMaxBwGbps float64 = 0
	var totalOutBwGbps float64 = 0
	var specialMaxBw float64 = 0

	var mikuLowRatioNodeCnt = 0

	machinelist := ""

	customerMap := make(map[uint32]int)

	logger := zlog.Logger
	logger = logger.Level(3)

	for _, node := range s.allNodesMap {
		if !node.IsDynamic {
			continue
		}
		s.TotalDynNoeCnt++
		if !util.CheckNodeUsable(logger, node, consts.TypeLive) {
			//log.Printf("checkNode nodeId:%s, machineId:%s check not pass, type: %s\n", node.Id, node.MachineId, node.ResourceType)
			s.NodeUnavailableCnt++
			continue
		}
		if !checkDynamicNodesPort(node) {
			s.NodeNoPortsCnt++
			continue
		}
		s.AvailableDynNodeCnt++
		if !checkCanScheduleOfTimeLimit(node, 3600) {
			s.TimeLimitCnt++
			continue
		}
		if node.IsBanTransProv {
			s.BanTransProvNodeCnt++
			//continue
		}
		for _, customerId := range node.CustomerIds {
			customerMap[customerId]++
		}
		if !ContainInIntSlice(1380460970, node.CustomerIds) {
			continue
		}
		if node.MachineId != "a71163883d96c375deef28ef8612b242" {
			continue
		}
		s.AvailableDynNodeAfterTimeLimitCnt++
		machinelist += node.MachineId + "\n"
		var totalMikuBw float64 = 0
		var nodeMaxOutBw float64 = 0
		var nodeOutBw float64 = 0
		for _, ipInfo := range node.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				s.PrivateIpCnt++
				continue
			}
			rawTotalMaxBwGbps += ipInfo.MaxOutMBps * 8 / 1000
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
			if ContainInStringSlice(node.MachineId, SepcialNodeList) {
				specialMaxBw += ipInfo.MaxInMBps * 8 / 1000
			}
			s.AvailableIpCnt++
			totalMaxBwGbps += ipInfo.MaxOutMBps * 8
			totalOutBwGbps += ipInfo.OutMBps * 8
			nodeMaxOutBw += ipInfo.MaxOutMBps * 8
			nodeOutBw += ipInfo.OutMBps * 8
			//totalMaxBwGbps += ipInfo.MaxOutMBps * 8 / 1000
			//totalOutBwGbps += ipInfo.OutMBps * 8 / 1000
			//totalBwGbps += ipInfo.MaxOutMBps
		}

		nodeStreams := s.nodeStremasMap[node.Id]
		for _, streamInfo := range nodeStreams.Streams {
			for _, player := range streamInfo.Players {
				for _, ipInfo := range player.Ips {
					totalMikuBw += float64(ipInfo.Bandwidth * 8 / 1000000)
					fmt.Println(player.Protocol, float64(ipInfo.Bandwidth*8)/1000000, ipInfo.Ip, streamInfo.Key)
				}
			}
		}
		ratio := totalMikuBw / nodeOutBw
		if ratio < 0.3 && nodeMaxOutBw > 9000 {
			log.Println(node.Id, node.MachineId, totalMikuBw, nodeMaxOutBw, nodeOutBw)
			mikuLowRatioNodeCnt++
		}
	}

	fmt.Printf(`totalMaxBwGbps: %.0fGbps
mikuLowRatioNodeCnt: %d
rawTotalMaxBwGbps: %.0fMbps
totalOutBwGbps: %.0fMbps
specialMaxBw: %.0fGbps
NodeUnavailableCnt: %d
NodeNoPortsCnt: %d
PrivateIpCnt: %d
NetProbeStateErrIpCnt: %d
NetProbeStateErrIpCnt: %d
IpV6Cnt: %d
TotalDynNoeCnt: %d
AvailableDynNodeCnt: %d
AvailableDynNodeAfterTimeLimitCnt: %d
AvailableIpCnt: %d
BanTransProvNodeCnt: %d
timelimitCnt: %d
`,
		totalMaxBwGbps,
		mikuLowRatioNodeCnt,
		rawTotalMaxBwGbps,
		totalOutBwGbps,
		specialMaxBw,
		s.NodeUnavailableCnt,
		s.NodeNoPortsCnt,
		s.PrivateIpCnt,
		s.NetProbeStateErrIpCnt,
		s.NetProbeSpeedErrIpCnt,
		s.IpV6Cnt,
		s.TotalDynNoeCnt,
		s.AvailableDynNodeCnt,
		s.AvailableDynNodeAfterTimeLimitCnt,
		s.AvailableIpCnt,
		s.BanTransProvNodeCnt,
		s.TimeLimitCnt)

	log.Println("customer info:")
	for cursomerId, cnt := range customerMap {
		fmt.Println(cursomerId, CustomerIdMap[cursomerId], cnt)
	}

	file := fmt.Sprintf("machinelist-%d.csv", time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(machinelist), 0644)
	if err != nil {
		log.Println(err)
	}
	cmd := exec.Command("./qup", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

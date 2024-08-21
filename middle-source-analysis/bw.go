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
		s.AvailableDynNodeAfterTimeLimitCnt++
		machinelist += node.MachineId + "\n"
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
			totalMaxBwGbps += ipInfo.MaxOutMBps * 8 / 1000
			totalOutBwGbps += ipInfo.OutMBps * 8 / 1000
			//totalBwGbps += ipInfo.MaxOutMBps
		}
	}
	fmt.Printf(`totalMaxBwGbps: %.0fGbps
rawTotalMaxBwGbps: %.0fGbps
totalOutBwGbps: %.0fGbps
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

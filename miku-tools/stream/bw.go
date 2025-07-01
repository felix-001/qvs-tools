package stream

import (
	"fmt"
	"io/ioutil"
	"log"
	"middle-source-analysis/node"
	"middle-source-analysis/public"
	"os/exec"
	"time"

	localUtil "middle-source-analysis/util"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	zlog "github.com/rs/zerolog/log"
)

const (
	TotalMaxBwGbps                  = "totalMaxBwGbps"
	TotalOutBwGbps                  = "totalOutBwGbps"
	TotalAvailableBwGbps            = "totalAvailableBwGbps"
	RawTotalMaxBwGbps               = "rawTotalMaxBwGbps"
	RawTotalMaxBwGbpsAferIpv6       = "rawTotalMaxBwGbpsAferIpv6"
	RawTotalMaxBwGbpsAferProbeState = "rawTotalMaxBwGbpsAferProbeState"
	RawTotalMaxBwGbpsAferProbeSpeed = "rawTotalMaxBwGbpsAferProbeSpeed"
	SpecialNodesMaxBwGbps           = "specialNodesMaxBwGbps"
	MikuTotalOutBwGbps              = "mikuTotalOutBwGbps"

	MikuLowUtilizationNodeCnt         = "mikuLowUtilizationNodeCnt"
	NodeUnavailableCnt                = "nodeUnavailableCnt"
	NodeNoPortsCnt                    = "nodeNoPortsCnt"
	TotalDynNodeCnt                   = "totalDynNodeCnt"
	AvailableDynNodeCnt               = "availableDynNodeCnt"
	TimeLimitNodeCnt                  = "timeLimitNodeCnt"
	BanTransProvNodeCnt               = "banTransProvNodeCnt"
	AvailableDynNodeAfterTimeLimitCnt = "availableDynNodeAfterTimeLimitCnt"
	IpV6Cnt                           = "ipV6Cnt"
	PrivateIpCnt                      = "privateIpCnt"
	NetProbeStateErrIpCnt             = "netProbeStateErrIpCnt"
	NetProbeSpeedErrIpCnt             = "netProbeSpeedErrIpCnt"
	AvailableIpCnt                    = "availableIpCnt"
)

func getNodeReportBw(node *model.RtNode, bwMap map[string]float64) float64 {
	var reportBw float64
	nodeStreams := NodeStremasMap[node.Id]
	for _, streamInfo := range NodeStreamStreams {
		for _, player := range streamInfo.Players {
			for _, ipInfo := range player.Ips {
				bwMap[MikuTotalOutBwGbps] += float64(ipInfo.Bandwidth*8) / 1000000000.0
				reportBw += float64(ipInfo.Bandwidth * 8 / 1000000)
				//fmt.Println(player.Protocol, float64(ipInfo.Bandwidth*8)/1000000, ipInfo.Ip, streamInfo.Key)
			}
		}
	}
	return reportBw
}

func dumpBw(bwMap map[string]float64, counterMap map[string]int) {
	for k, bw := range bwMap {
		fmt.Printf("%s: %.0f\n", k, bw)
	}
	for k, cnt := range counterMap {
		fmt.Printf("%s: %d\n", k, cnt)
	}
}

func CalcTotalBw() {
	bwMap := make(map[string]float64)
	counterMap := make(map[string]int)
	customerMap := make(map[uint32]int)

	machinelist := ""

	logger := zlog.Logger
	logger = logger.Level(3)

	for _, node := range node.AllNodesMap {
		if !node.IsDynamic {
			continue
		}
		counterMap[TotalDynNodeCnt]++
		if !util.CheckNodeUsable(logger, node, consts.TypeLive) {
			//log.Printf("checkNode nodeId:%s, machineId:%s check not pass, type: %s\n", node.Id, node.MachineId, node.ResourceType)
			counterMap[NodeUnavailableCnt]++
			continue
		}
		if !localUtil.CheckDynamicNodesPort(node) {
			counterMap[NodeNoPortsCnt]++
			continue
		}
		counterMap[AvailableDynNodeCnt]++
		if !localUtil.CheckCanScheduleOfTimeLimit(node, 3600) {
			counterMap[TimeLimitNodeCnt]++
			continue
		}
		if node.IsBanTransProv {
			counterMap[BanTransProvNodeCnt]++
		}
		for _, customerId := range node.CustomerIds {
			customerMap[customerId]++
		}
		counterMap[AvailableDynNodeAfterTimeLimitCnt]++

		machinelist += node.MachineId + "\n"

		var nodeMaxOutBw float64 = 0
		var nodeOutBw float64 = 0

		for _, ipInfo := range node.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				counterMap[PrivateIpCnt]++
				continue
			}
			bwMap[RawTotalMaxBwGbps] += ipInfo.MaxOutMBps * 8 / 1000
			if ipInfo.IsIPv6 {
				counterMap[IpV6Cnt]++
				continue
			}
			bwMap[RawTotalMaxBwGbpsAferIpv6] += ipInfo.MaxOutMBps * 8 / 1000
			if ipInfo.IPStreamProbe.State != model.StreamProbeStateSuccess {
				counterMap[NetProbeStateErrIpCnt]++
				continue
			}
			bwMap[RawTotalMaxBwGbpsAferProbeState] += ipInfo.MaxOutMBps * 8 / 1000
			if ipInfo.IPStreamProbe.Speed < 8 && ipInfo.IPStreamProbe.MinSpeed < 6 {
				counterMap[NetProbeSpeedErrIpCnt]++
				continue
			}
			bwMap[RawTotalMaxBwGbpsAferProbeSpeed] += ipInfo.MaxOutMBps * 8 / 1000
			if localUtil.ContainInStringSlice(node.MachineId, public.SepcialNodeList) {
				bwMap[SpecialNodesMaxBwGbps] += ipInfo.MaxInMBps * 8 / 1000
			}
			counterMap[AvailableIpCnt]++
			bwMap[TotalMaxBwGbps] += ipInfo.MaxOutMBps * 8 / 1000
			bwMap[TotalOutBwGbps] += ipInfo.OutMBps * 8 / 1000
			nodeMaxOutBw += ipInfo.MaxOutMBps * 8
			nodeOutBw += ipInfo.OutMBps * 8
		}

		reportBw := getNodeReportBw(node, bwMap)
		ratio := reportBw / nodeOutBw
		if ratio < 0.3 {
			//log.Println(node.Id, node.MachineId, totalMikuBw, nodeMaxOutBw, nodeOutBw)
			counterMap[MikuLowUtilizationNodeCnt]++
		}
	}
	bwMap[TotalAvailableBwGbps] = bwMap[TotalMaxBwGbps]*0.8 - bwMap[TotalOutBwGbps]

	dumpBw(bwMap, counterMap)

	/*
		log.Println("customer info:")
		for cursomerId, cnt := range customerMap {
			fmt.Println(cursomerId, CustomerIdMap[cursomerId], cnt)
		}
	*/

	file := fmt.Sprintf("machinelist-%d.txt", time.Now().Unix())
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

func BwDis() {
	bwMap := make(map[string]uint64)
	for _, nodeStreams := range NodeStremasMap {
		if node, ok := node.AllNodesMap[nodeStreamNodeId]; ok && !node.IsDynamic {
			continue
		}
		for _, streamRT := range NodeStreamStreams {
			if streamRT.Bucket != Conf.Bucket {
				continue
			}
			for _, player := range streamRT.Players {
				for _, ipInfo := range player.Ips {
					if publicUtil.IsPrivateIP(ipInfo.Ip) {
						continue
					}
					isp, _, province := localUtil.GetLocate(ipInfo.Ip, IpParser)
					if isp == "" || province == "" {
						continue
					}
					key := isp + "_" + province
					bwMap[key] += ipInfo.Bandwidth
				}
			}
		}
	}
	var max uint64
	province := ""
	for key, bw := range bwMap {
		if max < bw {
			max = bw
			province = key
		}
		fmt.Println(key, bw)
	}
	log.Println("province:", province, "max:", max)
}

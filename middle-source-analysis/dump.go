package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"sort"
	"time"
)

var hdr = "流ID, 运营商, 大区, 在线人数, 边缘节点个数, ROOT节点个数, 放大比, 边缘节点详情, ROOT节点详情\n"

func (s *Parser) dump() {
	csv := hdr
	cnt := 0
	var totalBw float64
	var totalRelayBw float64

	roomMap := make(map[string][]string)
	roomOnlineMap := make(map[string]int)
	for streamId, streamDetail := range s.streamDetailMap {
		roomId, id := splitString(streamId)
		roomMap[roomId] = append(roomMap[roomId], id)
		cnt++

		for isp, detail := range streamDetail {
			for area, streamInfo := range detail {
				ratio := streamInfo.Bw / streamInfo.RelayBw
				csv += fmt.Sprintf("%s, %s, %s, %d, %d, %d, %.1f, %+v, %+v\n", streamId, isp, area,
					streamInfo.OnlineNum, len(streamInfo.EdgeNodes),
					len(streamInfo.RootNodes), ratio, streamInfo.EdgeNodes,
					streamInfo.RootNodes)
				totalBw += streamInfo.Bw
				totalRelayBw += streamInfo.RelayBw

				roomOnlineMap[roomId] += int(streamInfo.OnlineNum)
			}
		}

	}
	file := fmt.Sprintf("%d.csv", time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}

	/*
		log.Println("cnt:", cnt)
		log.Printf("totalBw: %.1f, totalRelayBw: %.1f, totalRatio: %.1f", totalBw,
			totalRelayBw, totalBw/totalRelayBw)
		log.Println("room count:", len(roomMap))
		for roomId, ids := range roomMap {
			fmt.Println(roomId, ids)
		}
		log.Println("room - onlineNum info")
		for roomId, onlineNum := range roomOnlineMap {
			fmt.Println(roomId, onlineNum)
		}
	*/
}

func str2unix(s string) (int64, error) {
	loc, _ := time.LoadLocation("Local")
	the_time, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return the_time.Unix(), nil
}

func (s *Parser) saveNodesStatusDetailToCsv(nodeUnavailableDetail map[string][]NodeUnavailableDetail, schedInfos []SchedInfo) {
	csv := "开始时间, 结束时间, 原因, 详细\n"
	for _, schedInfo := range schedInfos {
		csv += schedInfo.NodeId + "\n"
		details := nodeUnavailableDetail[schedInfo.NodeId]
		for _, detail := range details {
			csv += fmt.Sprintf("%s, %s, %s, %s\n", detail.Start, detail.End, detail.Reason, detail.Detail)
		}
	}
	file := fmt.Sprintf("%s-nodes-detail-%d.csv", s.conf.Stream, time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
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

func (s *Parser) dumpStream() {
	nodeUnavailableDetailMap := s.getNodeUnavailableDetail("./node_info/nodeinfo-20240806222739.json", "2024-08-07 00:00:00",
		"2024-08-08 00:00:00")
	streamSchedInfos := s.getStreamSchedInfos()
	sort.Slice(streamSchedInfos, func(i, j int) bool {
		return streamSchedInfos[i].StartTime < streamSchedInfos[j].StartTime
	})
	s.saveStreamSchedInfosToCsv(streamSchedInfos)
	s.saveNodesStatusDetailToCsv(nodeUnavailableDetailMap, streamSchedInfos)

	/*
		nodeDetailMap := make(map[string][]NodeUnavailableDetail)
		start := streamSchedInfos[0].StartTime / 1000
		end := streamSchedInfos[len(streamSchedInfos)-1].StartTime / 1000
		for _, schedInfo := range streamSchedInfos {
			if _, ok := nodeDetailMap[schedInfo.NodeId]; !ok {
				nodeDetailMap[schedInfo.NodeId] = make([]NodeUnavailableDetail, 0)
			}
			details := nodeUnavailableDetailMap[schedInfo.NodeId]
			log.Println("node unavailable details:", len(details))
			for _, detail := range details {
				detailStart, err := str2unix(detail.Start)
				if err != nil {
					log.Println(err)
					continue
				}
				detailEnd, err := str2unix(detail.End)
				if err != nil {
					log.Println(err)
					continue
				}
				if detailEnd > start && detailStart < end {
					detail.Detail = strings.ReplaceAll(detail.Detail, ",", " ")
					nodeDetailMap[schedInfo.NodeId] = append(nodeDetailMap[schedInfo.NodeId], detail)
				}
			}
		}

		s.saveNodeDetailToCsv(nodeDetailMap)
	*/

}

func (s *Parser) saveNodeDetailToCsv(nodeDetailMap map[string][]NodeUnavailableDetail) {
	csv := "节点, 开始时间, 结束时间, reason, detail\n"
	for nodeId, details := range nodeDetailMap {
		for _, detail := range details {
			csv += fmt.Sprintf("%s, %s, %s, %s, %s\n", nodeId, detail.Start,
				detail.End, detail.Reason, detail.Detail)
		}
	}
	file := fmt.Sprintf("%s-node-detail-%d.csv", s.conf.Stream, time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
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

func (s *Parser) saveStreamSchedInfosToCsv(streamSchedInfos []SchedInfo) {
	csv := "时间, ConnId, NodeId, MachineId\n"
	for _, schedInfo := range streamSchedInfos {
		timeStr := unixToTimeStr(schedInfo.StartTime / 1000)
		csv += fmt.Sprintf("%s, %s, %s, %s\n", timeStr, schedInfo.ConnId,
			schedInfo.NodeId, schedInfo.MachindId)
	}
	file := fmt.Sprintf("%s-nodes-%d.csv", s.conf.Stream, time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

func (s *Parser) dumpNodeStreams() {
	node := s.conf.Node
	for _, stream := range s.nodeStremasMap[node].Streams {
		fmt.Println("bucket:", stream.AppName, "stream:", stream.Key)
		for _, player := range stream.Players {
			fmt.Printf("\t%s\n", player.Protocol)
			for _, ipInfo := range player.Ips {
				fmt.Printf("\t\t ip: %s, onlineNum: %d, bw: %d\n", ipInfo.Ip, ipInfo.OnlineNum, ipInfo.Bandwidth)
			}
		}
		for _, pusher := range stream.Pusher {
			fmt.Println(pusher.ConnectId)
		}
	}
}

func TimeStrSub(s string, subVal int) string {
	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, s)
	if err != nil {
		fmt.Printf("解析时间失败: %v\n", err)
		return ""
	}
	t = t.Add(-time.Duration(subVal) * time.Second)
	return t.Format(layout)
}

func (s *Parser) loadNodePoint(file string) []NodeInfo {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln("read fail", file, err)
		return nil
	}
	nodeInfos := make([]NodeInfo, 0)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		var nodeInfo NodeInfo
		if err := json.Unmarshal([]byte(line), &nodeInfo); err != nil {
			log.Println(err)
			continue
		}
		nodeInfos = append(nodeInfos, nodeInfo)
	}
	log.Println("nodeInfos cnt:", len(nodeInfos))
	return nodeInfos
}

func (s *Parser) isNodeAvailable(nodeInfo *NodeInfo) bool {
	if nodeInfo.RuntimeStatus != "Serving" {
		return false
	}
	if !nodeInfo.StreamdPorts {
		return false
	}
	if !nodeInfo.HaveAvailableIp {
		return false
	}
	return true
}

func (s *Parser) getUnavailableReasion(nodeInfo *NodeInfo) string {
	if nodeInfo.RuntimeStatus != "Serving" {
		return "offline"
	}
	if !nodeInfo.StreamdPorts {
		return "no streamd ports"
	}
	if !nodeInfo.HaveAvailableIp {
		return "no available ip"
	}
	return "ok"
}

func (s *Parser) buildNodeUnavailableDetailMap(nodeInfos []NodeInfo, nodeUnavailableDetailMap map[string][]NodeUnavailableDetail) {
	//midnight := getMidnight2()
	lastNodeInfoMap := make(map[string]*NodeInfo)
	for _, nodeInfo := range nodeInfos {
		last, ok := lastNodeInfoMap[nodeInfo.NodeId]
		if !ok || last == nil {
			info := nodeInfo
			lastNodeInfoMap[nodeInfo.NodeId] = &info
			continue
		}

		if _, ok := nodeUnavailableDetailMap[nodeInfo.NodeId]; !ok {
			nodeUnavailableDetailMap[nodeInfo.NodeId] = make([]NodeUnavailableDetail, 0)
		}
		reason := s.getUnavailableReasion(last)
		bytes, err := json.Marshal(last.ErrIps)
		if err != nil {
			log.Println(err)
			continue
		}
		detail := strings.ReplaceAll(string(bytes), ",", " ")
		nodeUnavailableDetailMap[nodeInfo.NodeId] = append(nodeUnavailableDetailMap[nodeInfo.NodeId],
			NodeUnavailableDetail{
				Start:    nodeInfo.StartTime,
				End:      nodeInfo.EndTime,
				Reason:   reason,
				Detail:   detail,
				Duration: nodeInfo.Duration,
			})

		lastNodeInfoMap[nodeInfo.NodeId] = nil
	}
}

func (s *Parser) getDuration(start, end string) time.Duration {
	layout := "2006-01-02 15:04:05"
	t1, err1 := time.Parse(layout, start)
	t2, err2 := time.Parse(layout, end)

	if err1 != nil || err2 != nil {
		fmt.Println("Error parsing time:", err1, err2)
		return 0
	}

	// 计算两个时间点之间的持续时间
	duration := t2.Sub(t1)
	return duration
}

func (s *Parser) getNodeUnavailableDetail(days int) map[string][]NodeUnavailableDetail {
	cur := time.Now().Format("2006_01_02")
	dates := generateDateRange(cur, days)
	nodeUnavailableDetailMap := make(map[string][]NodeUnavailableDetail)
	for _, filename := range dates {
		filename = path + "/" + filename
		nodeInfos := s.loadNodePoint(filename)
		s.buildNodeUnavailableDetailMap(nodeInfos, nodeUnavailableDetailMap)
	}
	return nodeUnavailableDetailMap
}

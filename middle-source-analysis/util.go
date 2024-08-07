package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
)

func getLocate(ip string, ipParser *ipdb.City) (string, string, string) {
	locate, err := ipParser.Find(ip)
	if err != nil {
		log.Println(err)
		return "", "", ""
	}
	if locate.Isp == "" {
		//log.Println("country", locate.Country, "isp", locate.Isp, "city", locate.City, "region", locate.Region, "ip", ip)
	}
	area := monitorUtil.ProvinceAreaRelation(locate.Region)
	return locate.Isp, area, locate.Region
}

func getNodeLocate(node *model.RtNode, ipParser *ipdb.City) (string, string) {
	for _, ip := range node.Ips {
		if ip.IsIPv6 {
			continue
		}
		if publicUtil.IsPrivateIP(ip.Ip) {
			continue
		}
		isp, area, _ := getLocate(ip.Ip, ipParser)
		if area != "" {
			return isp, area
		}
	}
	return "", ""
}

func (s *Parser) isRoot(node *model.RtNode) bool {
	_, ok := s.allRootNodesMapByNodeId[node.Id]
	return ok
}

func ContainInStringSlice(target string, slice []string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}

	return false
}

func getIpAreaIsp(ipParser *ipdb.City, ip string) (string, string, error) {
	locate, err := ipParser.Find(ip)
	if err != nil {
		return "", "", err
	}
	areaIspKey, _ := util.GetAreaIspKey(locate)
	parts := strings.Split(areaIspKey, "_")
	if len(parts) != 5 {
		return "", "", fmt.Errorf("parse areaIspKey err, %s", areaIspKey)
	}
	area := parts[3]
	isp := parts[4]
	if area == "" {
		return "", "", fmt.Errorf("area empty")
	}
	if isp == "" {
		return "", "", fmt.Errorf("isp empty")
	}
	return area, isp, nil
}

func splitString(s string) (string, string) {
	// 从左到右遍历，找到最后一个数字的位置
	var lastDigitIndex int
	for i, char := range s {
		if !unicode.IsDigit(char) {
			lastDigitIndex = i
			break
		}
	}

	// 根据最后一个数字的位置分割字符串
	part1, part2 := s[:lastDigitIndex], s[lastDigitIndex:]

	return part1, part2
}

func convertMbps(bw uint64) float64 {
	return float64(bw) * 8 / 1e6
}

func (s *Parser) getStreamDetail(stream *model.StreamInfoRT) (int, float64) {
	totalOnlineNum := 0
	var totalBw float64
	for _, player := range stream.Players {
		for _, ipInfo := range player.Ips {
			totalOnlineNum += int(ipInfo.OnlineNum)
			totalBw += convertMbps(ipInfo.Bandwidth)
		}
	}
	return totalOnlineNum, totalBw
}

func (s *Parser) calcRelayBw(streamDetail map[string]map[string]*StreamInfo, stream *model.StreamInfoRT, node *model.RtNode) {
	for _, detail := range streamDetail {
		for _, streamInfo := range detail {
			streamInfo.RelayBw += convertMbps(stream.RelayBandwidth)
		}
	}
}

func (s *Parser) getNodeOnlineNum(streamInfo *model.StreamInfoRT) int {
	totalOnlineNum := 0
	for _, player := range streamInfo.Players {
		for _, ipInfo := range player.Ips {
			totalOnlineNum += int(ipInfo.OnlineNum)
			log.Println("protocol:", player.Protocol)
		}
	}
	return totalOnlineNum
}

func unixToTimeStr(t int64) string {
	timestamp := int64(t)
	timeObj := time.Unix(timestamp, 0)
	formattedTime := timeObj.Format(time.DateTime)
	return formattedTime
}

// TODO
func (s *Parser) isNodeUsable(node *model.RtNode) bool {
	return true
}

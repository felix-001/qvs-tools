package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func findLatestFile(dir string) (string, error) {
	var latestFile string
	latestTime := time.Unix(0, 0) // 初始化为 Unix 纪元时间，即1970-01-01 00:00:00 +0000 UTC

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略目录
		if info.IsDir() {
			return nil
		}

		// 检查文件修改时间，并更新最新的文件信息
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = path
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if latestFile == "" {
		return "", fmt.Errorf("no files found in directory")
	}

	return latestFile, nil
}

// generateDateRange 生成一个日期范围的切片
func generateDateRange(date string, days int) []string {
	// 解析日期字符串
	parsedDate, err := time.Parse("2006_01_02", date)
	if err != nil {
		fmt.Println("Error parsing date:", err)
		return nil
	}

	// 创建一个切片来存储日期
	dateRange := make([]string, 0, days)

	// 计算日期范围并填充切片
	for i := 0; i < days; i++ {
		dateStr := parsedDate.AddDate(0, 0, -i).Format("2006_01_02")
		dateRange = append(dateRange, dateStr)
	}

	// 因为我们需要从当前日期到（当前日期 - days + 1），所以返回翻转的切片
	for i, j := 0, len(dateRange)-1; i < j; i, j = i+1, j-1 {
		dateRange[i], dateRange[j] = dateRange[j], dateRange[i]
	}

	return dateRange
}

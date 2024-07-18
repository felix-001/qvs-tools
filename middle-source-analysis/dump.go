package main

import (
	"fmt"
	"io/ioutil"
	"log"
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

package main

import "fmt"

func (s *Parser) dumpNodeStreams(node string) {
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

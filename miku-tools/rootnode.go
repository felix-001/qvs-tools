package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/redis/go-redis/v9"
	zlog "github.com/rs/zerolog/log"
)

func (s *Parser) buildRootNodesMap() {
	dynamicRootNodesMap, err := GetDynamicRootNodes(s.RedisCli)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println("map len", len(dynamicRootNodesMap))
	s.allRootNodesMapByNodeId = make(map[string]*model.RtNode)
	s.allRootNodesMapByAreaIsp = dynamicRootNodesMap
	for _, rootNodes := range dynamicRootNodesMap {
		for _, rootNode := range rootNodes {
			node, ok := s.allNodesMap[rootNode.NodeId]
			if !ok {
				log.Println("not found root node in all nodes buf", rootNode.NodeId)
				continue
			}
			s.allRootNodesMapByNodeId[rootNode.NodeId] = node
		}
	}
}

func GetDynamicRootNodes(redisCli *redis.ClusterClient) (
	map[string][]*DynamicRootNode, error) {

	dynamicRootNodesMap := make(map[string][]*DynamicRootNode)
	ctx := context.Background()
	res, err := redisCli.HGetAll(ctx, "miku_dynamic_root_nodes_map").Result()
	if err != nil {
		log.Println(err)
		return dynamicRootNodesMap, consts.ErrRedisHGetAll
	}

	for areaIsp, value := range res {
		var nodes []*DynamicRootNode
		if err = json.Unmarshal([]byte(value), &nodes); err != nil {
			log.Println(err)
			continue
		}

		dynamicRootNodesMap[areaIsp] = nodes
	}

	return dynamicRootNodesMap, nil
}

func (s *Parser) getStreamNodes(sid, bkt string) map[string][]*model.RtNode {
	nodeMap := make(map[string][]*model.RtNode)
	for areaIsp, nodes := range s.allRootNodesMapByAreaIsp {
		for _, rootNode := range nodes {
			report := s.nodeStremasMap[rootNode.NodeId]
			if time.Now().Unix()-report.LastUpdateTime > 300 {
				log.Println(rootNode.NodeId, "stream offline", time.Unix(report.LastUpdateTime, 0).Format("2006-01-02 15:04:05 -0700 MST"))
				continue
			}
			for _, stream := range report.Streams {
				if stream.Bucket == bkt && stream.Key == sid {
					onlineNum := s.getNodeOnlineNum(stream)
					log.Println("node:", rootNode.NodeId, "onlineNum:", onlineNum,
						"bandwidth:", stream.Bandwidth, "relayBandWidth", stream.RelayBandwidth,
						"relayType:", stream.RelayType)
					node, ok := s.allNodesMap[rootNode.NodeId]
					if !ok {
						log.Println("node", rootNode.NodeId, "not found in all nodes buf")
						continue
					}
					nodeMap[areaIsp] = append(nodeMap[areaIsp], node)
					break
				}
			}
		}
	}
	return nodeMap
}

// TODO: 节点的每个网卡的带宽利用率, 这里的出口带宽不能使用streamreport上报的，其他业务线，
// 或者别的bucket下的流也会使用这个出口带宽
func (s *Parser) getNodeDetailMap(streamDetail map[string]map[string]*StreamInfo,
	stream *model.StreamInfoRT, node *model.RtNode) {

	for _, player := range stream.Players {
		/*
			if len(player.Ips) > 1 {
				log.Println("player ips > 1", node.Id)
			}
		*/
		var lastIsp, lastArea string
		for _, ipInfo := range player.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				//log.Println("private ip", ipInfo.Ip)
				continue
			}
			if ipInfo.Ip == "" {
				//log.Println("ip empty")
				continue
			}
			area, isp, err := getIpAreaIsp(s.IpParser, ipInfo.Ip)
			if err != nil {
				log.Println("getIpAreaIsp err", ipInfo.Ip, err)
				continue
			}
			if lastIsp != "" && lastIsp != isp {
				log.Println("isp not equal", node.Id, stream.Key, "last:", lastIsp, "cur:", isp)
			}
			if lastArea != "" && lastArea != area {
				log.Println("area not equal", node.Id, stream.Key, "last:", lastArea, "cur:", area)
			}
			if lastIsp == "" {
				lastIsp = isp
			}
			if lastArea == "" {
				lastArea = area
			}
			if _, ok := streamDetail[isp]; !ok {
				streamDetail[isp] = make(map[string]*StreamInfo)
			}
			if _, ok := streamDetail[isp][area]; !ok {
				streamDetail[isp][area] = &StreamInfo{
					//RelayType: stream.RelayType,
					//Protocol:  player.Protocol,
					//RelayBw:   convertMbps(stream.RelayBandwidth),
					OnlineNum: ipInfo.OnlineNum,
					Bw:        convertMbps(ipInfo.Bandwidth),
				}
			}
			streamDetail[isp][area].OnlineNum += ipInfo.OnlineNum
			streamDetail[isp][area].Bw += float64(ipInfo.Bandwidth)
		}
	}
}

func (s *Parser) checkNodeStreamIpLocate(stream *model.StreamInfoRT, node *model.RtNode) {
	for _, player := range stream.Players {
		var lastIsp, lastArea string
		for _, ipInfo := range player.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				//log.Println("private ip", ipInfo.Ip)
				continue
			}
			if ipInfo.Ip == "" {
				//log.Println("ip empty")
				continue
			}
			area, isp, err := getIpAreaIsp(s.IpParser, ipInfo.Ip)
			if err != nil {
				log.Println("getIpAreaIsp err", ipInfo.Ip, err)
				continue
			}
			if lastIsp != "" && lastIsp != isp {
				log.Println("isp not equal", node.Id, stream.Key, "last:", lastIsp, "cur:", isp)
			}
			if lastArea != "" && lastArea != area {
				log.Println("area not equal", node.Id, stream.Key, "last:", lastArea, "cur:", area)
			}
			if lastIsp == "" {
				lastIsp = isp
			}
			if lastArea == "" {
				lastArea = area
			}
		}
	}
}

func (s *Parser) check(node *model.RtNode, streamInfo *model.NodeStreamInfo) bool {
	if !s.checkNode(node) {
		log.Println("check node status err", node.Id)
		return false
	}
	if !node.IsDynamic {
		return false
	}
	if streamInfo == nil {
		//log.Println(node.Id, "not found stream info")
		return false
	}
	if time.Now().Unix()-streamInfo.LastUpdateTime > 300 {
		/*
			log.Println(node.Id, "stream offline",
				time.Unix(streamInfo.LastUpdateTime, 0).
					Format("2006-01-02 15:04:05 -0700 MST"))
		*/
		return false
	}
	if streamInfo.NodeId != node.Id {
		log.Println("check stream info node id err")
	}
	return true
}

func (s *Parser) checkNode(node *model.RtNode) bool {
	if node == nil || !node.IsDynamic || node.RuntimeStatus != model.StateServing {
		return false
	}

	// 检查节点能力：状态、ability、service
	if !util.CheckNodeUsable(zlog.Logger, node, consts.TypeLive) {
		log.Printf("checkNode nodeId:%s, machineId:%s check not pass, type: %s\n",
			node.Id, node.MachineId, node.ResourceType)
		return false
	}

	if node.StreamdPorts.Http <= 0 || node.StreamdPorts.Https <= 0 || node.StreamdPorts.Wt <= 0 {
		log.Printf("getAllDynamicNodesReport check http,https,wt port failed, nodeId:%s\n", node.Id)
		return false
	}
	return true
}

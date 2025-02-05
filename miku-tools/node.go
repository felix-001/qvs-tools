package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	public "github.com/qbox/mikud-live/common/model"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func (s *Parser) GetNodeByIp() {
	for _, node := range s.allNodesMap {
		for _, ipInfo := range node.Ips {
			if ipInfo.Ip == s.conf.Ip {
				_, ok := s.allRootNodesMapByNodeId[node.Id]
				fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "isRoot:", ok)
				break
			}
		}
	}
}

func (s *Parser) NodeDis() {
	areaMap := make(map[string]bool)
	for _, node := range s.allNodesMap {
		if node.RuntimeStatus != "Serving" {
			continue
		}
		if node.StreamdPorts.Http == 0 {
			continue
		}
		if node.StreamdPorts.Rtmp == 0 {
			continue
		}
		if !util.CheckNodeUsable(zlog.Logger, node, consts.TypeLive) {
			continue
		}
		isp, area, _ := getNodeLocate(node, s.IpParser)
		areaMap[area+isp] = true

	}

	needAreas := make([]string, 0)
	for _, area := range Areas {
		for _, isp := range Isps {
			areaIsp := area + isp
			if _, ok := areaMap[areaIsp]; !ok {
				needAreas = append(needAreas, areaIsp)
			}
		}
	}
	bytes, err := json.MarshalIndent(areaMap, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
	fmt.Println("needAreas:", needAreas)
}

type NodeCallback interface {
	OnNode(node *public.RtNode)
	OnIp(node *public.RtNode, ip *public.RtIpStatus)
}

type NodeTraverse struct {
	RedisAddrs []string
	logger     zerolog.Logger
	cb         NodeCallback
}

func NewNodeTraverse(logger zerolog.Logger, cb NodeCallback, RedisAddrs []string) *NodeTraverse {
	return &NodeTraverse{
		logger:     logger,
		cb:         cb,
		RedisAddrs: RedisAddrs,
	}
}

func (n *NodeTraverse) Traverse() {
	fmt.Println("NodeTraverse")
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      n.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		zlog.Fatal().Err(err).Msg("")
	}

	allNodes, err := public.GetAllRTNodes(n.logger, redisCli)
	if err != nil {
		n.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	for _, node := range allNodes {
		n.cb.OnNode(node)
		for _, ip := range node.Ips {
			n.cb.OnIp(node, &ip)
		}
	}
}

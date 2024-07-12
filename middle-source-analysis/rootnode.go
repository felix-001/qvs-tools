package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/common/model"
	"github.com/redis/go-redis/v9"
)

func (s *Parser) buildRootNodesMap() {
	dynamicRootNodesMap, err := GetDynamicRootNodes(s.redisCli)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("map len", len(dynamicRootNodesMap))
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

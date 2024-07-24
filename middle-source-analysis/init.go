package main

import (
	"log"

	"github.com/qbox/mikud-live/cmd/sched/dal"
	"github.com/qbox/mikud-live/common/model"
)

func (s *Parser) buildAllNodesMap() {
	allNodes, err := dal.GetAllNode(s.redisCli)
	if err != nil {
		log.Fatalln(err)
	}
	allNodesMap := make(map[string]*model.RtNode)
	for _, node := range allNodes {
		allNodesMap[node.Id] = node
	}
	s.allNodesMap = allNodesMap
}

func (s *Parser) buildNodeStreamsMap() {
	nodeStreamsMap := make(map[string]*model.NodeStreamInfo)
	for nodeId := range s.allNodesMap {
		node := s.allNodesMap[nodeId]
		if node == nil {
			continue
		}
		if !node.IsDynamic {
			continue
		}
		report, err := s.getNodeAllStreams(nodeId)
		if err != nil || report == nil {
			continue
		}
		nodeStreamsMap[nodeId] = report
	}
	s.nodeStremasMap = nodeStreamsMap
	log.Println("nodeStremasMap len", len(s.nodeStremasMap))
}

func (s *Parser) init() {
	s.buildAllNodesMap()
	s.buildNodeStreamsMap()
	s.buildRootNodesMap()
}

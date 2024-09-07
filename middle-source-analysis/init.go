package main

import (
	"fmt"
	"log"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/qbox/mikud-live/cmd/agent/common/util"
	"github.com/qbox/mikud-live/cmd/sched/dal"
	"github.com/qbox/mikud-live/common/model"
	"github.com/rs/zerolog"
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
	fmt.Println("all nodes count:", len(s.allNodesMap))
}

func (s *Parser) buildNodeStreamsMap() {
	nodeStreamsMap := make(map[string]*model.NodeStreamInfo)
	for nodeId := range s.allNodesMap {
		node := s.allNodesMap[nodeId]
		if node == nil {
			continue
		}
		/*
			if !node.IsDynamic {
				continue
			}
		*/
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
	if s.conf.NodeInfo {
		s.buildAllNodesMap()
		s.buildRootNodesMap()
	}
	if s.conf.NeedNodeStreamInfo {
		s.buildNodeStreamsMap()
	}
	if s.conf.Prometheus {
		prometheus.MustRegister(dynIpStatusMetric)
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.CallerMarshalFunc = util.LogShortPath
	s.logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05.000", NoColor: true}).With().Timestamp().Caller().Logger()
}

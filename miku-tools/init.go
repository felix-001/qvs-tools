package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	localUtil "middle-source-analysis/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/qbox/mikud-live/cmd/agent/common/util"
	"github.com/qbox/mikud-live/common/model"
	public "github.com/qbox/mikud-live/common/model"
	"github.com/rs/zerolog"
)

func (s *Parser) buildAllNodesMap() {
	fmt.Println("buildAllNodesMap")
	if _, err := os.Stat("/tmp/allNodes.json"); err == nil {
		// 文件存在，从文件加载节点信息
		file, err := os.ReadFile("/tmp/allNodes.json")
		if err != nil {
			s.logger.Error().Msgf("读取/tmp/allNodes.json文件失败: %+v", err)
			return
		}
		if err := json.Unmarshal(file, &s.allNodesMap); err != nil {
			s.logger.Error().Msgf("解析/tmp/allNodes.json文件失败: %+v", err)
			return
		}
		fmt.Println("从/tmp/allNodes.json文件加载节点信息成功")
		return
	}
	allNodes, err := public.GetAllRTNodes(s.logger, s.RedisCli)
	if err != nil {
		s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	allNodesMap := make(map[string]*model.RtNode)
	for _, node := range allNodes {
		allNodesMap[node.Id] = node
	}
	s.allNodesMap = allNodesMap
	//fmt.Println("all nodes count:", len(s.allNodesMap))
}

func (s *Parser) buildNodeStreamsMap() {
	nodeStreamsMap := make(map[string]*model.NodeStreamInfo)
	for nodeId := range s.allNodesMap {
		node := s.allNodesMap[nodeId]
		if node == nil {
			continue
		}
		report, err := localUtil.GetNodeAllStreams(nodeId, s.RedisCli)
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
		// TODO: 使用文件缓存+线上更新并行的方式
		s.buildAllNodesMap()
	}
	if s.conf.RootNodeInfo {
		s.buildRootNodesMap()
	}
	if s.conf.NeedNodeStreamInfo {
		s.buildNodeStreamsMap()
	}
	if s.conf.Prometheus {
		prometheus.MustRegister(localUtil.DynIpStatusMetric)
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.CallerMarshalFunc = util.LogShortPath
	s.logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05.000", NoColor: true}).With().Timestamp().Caller().Logger()
}

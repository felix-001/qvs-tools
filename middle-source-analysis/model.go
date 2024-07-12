package main

import (
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	RedisAddrs []string    `json:"redis_addrs"`
	IPDB       ipdb.Config `json:"ipdb"`
}

type Parser struct {
	redisCli                 *redis.ClusterClient
	ipParser                 *ipdb.City
	nodeStremasMap           map[string]*model.NodeStreamInfo
	allNodesMap              map[string]*model.RtNode
	allRootNodesMapByAreaIsp map[string][]*DynamicRootNode
	allRootNodesMapByNodeId  map[string]*model.RtNode
	// key1: streamId key2: isp key3: area
	streamDetailMap map[string]map[string]map[string]*StreamInfo
	needCheckNode   bool
}

type DynamicRootNode struct {
	NodeId        string
	Forbidden     bool
	Err           string
	ForbiddenTime int64
}

type NodeDetail struct {
	OnlineNum uint32
	RelayBw   float64
	Bw        float64
	MaxBw     float64
	Ratio     float64
	RelayType uint32
	Protocol  string
	NodeId    string
}

type StreamInfo struct {
	EdgeNodes []string
	RootNodes []string
	RelayBw   float64
	Bw        float64
	OnlineNum uint32
}

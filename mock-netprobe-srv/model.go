package main

import (
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

type NodeExtra struct {
	StreamInfo       model.NodeStreamInfo
	LowThresholdTime int64
}

type Config struct {
	RedisAddrs    []string    `json:"redis_addrs"`
	NodesDataFile string      `json:"nodes_data_file"`
	IPDB          ipdb.Config `json:"ipdb"`
}

type NetprobeSrv struct {
	redisCli   *redis.ClusterClient
	conf       Config
	nodes      []*model.RtNode
	ipParser   *ipdb.City
	nodeExtras map[string]*NodeExtra
	// key1: bucket key2: stream key3: node key4: ip
	streamReportMap map[string]map[string]map[string]map[string]int
}

type Pair struct {
	Key string
	Val float64
}

type Router struct {
	Path    string
	Params  []string
	Handler func(paramMap map[string]string) string
}

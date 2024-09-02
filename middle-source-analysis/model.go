package main

import (
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

type CkConfig struct {
	Host   []string `json:"host"`
	DB     string   `json:"db"`
	User   string   `json:"user"`
	Passwd string   `json:"passwd"`
	Table  string   `json:"table"`
}

type Config struct {
	RedisAddrs         []string    `json:"redis_addrs"`
	IPDB               ipdb.Config `json:"ipdb"`
	CK                 CkConfig    `json:"ck"`
	Bucket             string
	Stream             string
	Node               string
	CheckNode          bool
	Monitor            bool
	NeedIpParer        bool
	NeedCk             bool
	Streams            bool
	PrometheusAddr     string
	NeedNodeStreamInfo bool
	Bw                 bool
	LagFile            string
	NodeInfo           bool
	Prometheus         bool
	Redis              bool
	Pcdn               bool
	DnsResFile         string
	PathqueryLogFile   string
	StopAllStream      bool
	Ak                 string
	Sk                 string
}

type Parser struct {
	redisCli                 *redis.ClusterClient
	ipParser                 *ipdb.City
	nodeStremasMap           map[string]*model.NodeStreamInfo
	allNodesMap              map[string]*model.RtNode
	allRootNodesMapByAreaIsp map[string][]*DynamicRootNode
	allRootNodesMapByNodeId  map[string]*model.RtNode
	allNodeInfoMap           map[string]*NodeInfo
	// key1: streamId key2: isp key3: area
	streamDetailMap                   map[string]map[string]map[string]*StreamInfo
	file                              *os.File
	ck                                driver.Conn
	conf                              *Config
	streamInfoMap                     map[string]map[string]*StreamInfo
	NodeUnavailableCnt                int
	NodeNoPortsCnt                    int
	PrivateIpCnt                      int
	NetProbeStateErrIpCnt             int
	NetProbeSpeedErrIpCnt             int
	IpV6Cnt                           int
	TimeLimitCnt                      int
	TotalDynNoeCnt                    int
	AvailableDynNodeCnt               int
	AvailableDynNodeAfterTimeLimitCnt int
	AvailableIpCnt                    int
	BanTransProvNodeCnt               int
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
	EdgeNodes    []string
	RootNodes    []string
	OfflineNodes []string
	RelayBw      float64
	Bw           float64
	OnlineNum    uint32
}

type NodeUnavailableDetail struct {
	Start    string
	End      string
	Duration string
	Reason   string
	Detail   string
}

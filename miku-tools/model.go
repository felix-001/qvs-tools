package main

import (
	"middle-source-analysis/config"
	"middle-source-analysis/public"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"

	//"github.com/qbox/linking/internal/qvs.v1"

	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Parser struct {
	RedisCli                 *redis.ClusterClient
	IpParser                 *ipdb.City
	nodeStremasMap           map[string]*model.NodeStreamInfo
	allNodesMap              map[string]*model.RtNode
	allRootNodesMapByAreaIsp map[string][]*public.DynamicRootNode
	allRootNodesMapByNodeId  map[string]*model.RtNode
	allNodeInfoMap           map[string]*NodeInfo
	// key1: streamId key2: isp key3: area
	streamDetailMap                   map[string]map[string]map[string]*public.StreamInfo
	file                              *os.File
	CK                                driver.Conn
	conf                              *config.Config
	streamInfoMap                     map[string]map[string]*public.StreamInfo
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
	logger                            zerolog.Logger
	CmdMap                            map[string]public.CmdInfo
	appg                              appg.Client
}

package main

import (
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"

	//"github.com/qbox/linking/internal/qvs.v1"
	"github.com/qbox/mikud-live/cmd/dnspod/config"
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type CkConfig struct {
	Host   []string `json:"host"`
	DB     string   `json:"db"`
	User   string   `json:"user"`
	Passwd string   `json:"passwd"`
	Table  string   `json:"table"`
}

type Config struct {
	RedisAddrs     []string                      `json:"redis_addrs"`
	IPDB           ipdb.Config                   `json:"ipdb"`
	CK             CkConfig                      `json:"ck"`
	Secret         string                        `json:"secret"`
	PrometheusAddr string                        `json:"prometheus"`
	DyApiSecret    string                        `json:"dy_api_secret"`
	DyApiDomain    string                        `json:"dy_api_domain"`
	AccountCfg     qconfapi.Config               `json:"acc"`
	OriginKey      string                        `json:"origin_key"`
	OriginKeyDy    string                        `json:"origin_key_dy"`
	OriginKeyHw    string                        `json:"origin_key_hw"`
	NiulinkPath    string                        `json:"niulink_path"`
	KubeCfg        string                        `json:"kube_cfg"`
	DnsPod         config.DnspodConfig           `json:"dnspod"`
	SendKey        string                        `json:"send_key"`
	BwRatioConfig  map[string]map[string]float64 `json:"bw_ratio_config"` // key1: 大区 key2: isp(取值 移动/电信/联通/total)
	IdcBwConfig    map[string]map[string]int     `json:"idc_bw_config"`   // key1: idc key2: isp
	//Kodo               qvs.KODOConfig                `json:"kodo`
	Bucket             string
	SubCmd             string
	Stream             string
	Format             string
	Node               string
	NeedIpParer        bool
	NeedCk             bool
	NeedNodeStreamInfo bool
	LagFile            string
	NodeInfo           bool
	RootNodeInfo       bool
	Prometheus         bool
	Redis              bool
	DnsResFile         string
	PathqueryLogFile   string
	Ak                 string `json:"ak"`
	Sk                 string `json:"sk"`
	AdminAk            string `json:"admin_ak"`
	AdminSk            string `json:"admin_sk"`
	Cmd                string
	QosFile            string
	Help               bool
	Isp                string
	Province           string
	Area               string
	Pcdn               string
	Ip                 string
	T                  string
	Query              string
	N                  int
	QpmFile            string
	PcdnErr            string
	User               string
	ConnId             string
	Https              bool
	Interval           int
	SchedIp            string
	Uid                string
	Domain             string
	SourceId           string
	Origin             string
	Basesub            int
	SubStream          int
	Startid            int
	F                  string
	Force              bool
	RecordId           string
	Name               string
	Cnt                int
	TotoalNeedBw       int
	Method             string
	Body               string
	Addr               string
	Port               int
	ID                 string
	Transport          string
	Passwd             string
	Key                string
	Ns                 string
}

type CmdHandler func()

type CmdInfo struct {
	Handler CmdHandler
	Usage   string
	Depends []*bool
}

type Parser struct {
	RedisCli                 *redis.ClusterClient
	IpParser                 *ipdb.City
	nodeStremasMap           map[string]*model.NodeStreamInfo
	allNodesMap              map[string]*model.RtNode
	allRootNodesMapByAreaIsp map[string][]*DynamicRootNode
	allRootNodesMapByNodeId  map[string]*model.RtNode
	allNodeInfoMap           map[string]*NodeInfo
	// key1: streamId key2: isp key3: area
	streamDetailMap                   map[string]map[string]map[string]*StreamInfo
	file                              *os.File
	CK                                driver.Conn
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
	logger                            zerolog.Logger
	CmdMap                            map[string]CmdInfo
	appg                              appg.Client
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
	//EdgeNodes    []string
	//EdgeNodes    map[string]*model.StreamInfoRT
	//RootNodes    []string
	//OfflineNodes []string
	//StaticNodes []string
	RelayBw       float64
	Bw            float64
	OnlineNum     uint32
	NodeStreamMap map[string]map[string]*model.StreamInfoRT // key1: node type(edge/root/offline/static) key2: node Id
}

type NodeUnavailableDetail struct {
	Start    string
	End      string
	Duration string
	Reason   string
	Detail   string
}

type StreamDetail struct {
	model.StreamInfoRT
	NodeId string
}

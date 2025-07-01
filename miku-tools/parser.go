package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"middle-source-analysis/config"
	"middle-source-analysis/mock"
	"middle-source-analysis/public"
	"middle-source-analysis/util"
	localUtil "middle-source-analysis/util"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	//qlog "github.com/qbox/pili/base/qiniu/log.v1"
)

type Parser struct {
	RedisCli *redis.ClusterClient
	IpParser *ipdb.City
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
	CmdMap                            map[string]config.CmdInfo
	appg                              appg.Client
}

func dumpCmdMap(cmdMap map[string]config.CmdInfo) {
	for cmd, info := range cmdMap {
		fmt.Printf("%s\n\t%s\n", cmd, info.Usage)
	}
	fmt.Println()
	flag.PrintDefaults()
	os.Exit(0)
}

func newParser(conf *config.Config) *Parser {
	parser := &Parser{
		conf: conf,
	}
	cmdMap := map[string]config.CmdInfo{
		"hlschk": {
			Handler: parser.HlsChk,
			Usage:   "hls带宽统计",
		},
		"mockagent": {
			Handler: mock.MockAgent,
			Usage:   "mock agent, 主要为streamd能跑起来",
		},
		"streams": {
			Handler: parser.dumpStreams,
			Usage:   "dump每个流的放大比和每个流的详情",
		},
		"monitor": {
			Handler: parser.nodeMonitor,
			Usage:   "每隔一段时间,记录一下节点的变化信息(离线/没有ip可用/拉流探测失败),监控节点状态",
		},
		"nodesMonitor": {
			Handler: parser.rawNodesMonitor,
			Usage:   "每隔一段时间,记录dump一些所有节点信息",
			Depends: []*bool{&conf.Redis, &conf.NodeInfo},
		},
		"lag": {
			Handler: parser.LagAnalysis,
			Usage:   "分析从ck下载的streamd qos数据, 分析卡顿率高的原因",
			Depends: []*bool{&conf.NeedIpParer},
		},
		"loopplaycheck": {
			Handler: parser.LoopPlaycheck,
			Usage:   "302返点本省/本大区覆盖率检查,请求sched的playcheck接口",
			Depends: []*bool{&conf.NeedIpParer},
		},
		"playcheck": {
			Handler: parser.Playcheck,
			Usage:   "请求playcheck 302接口",
		},
		"dns": {
			Handler: parser.DnsChk,
			Usage:   "检查dns拨测的结果,看下权威dns解析是否符合预期",
		},
		"pathquerychk": {
			Handler: parser.pathqueryChk,
			Usage:   "分析从elk下载的pathquery日志文件,判断sched返回的回源路径是否符合预期",
		},
		"node": {
			Handler: parser.dumpNodeStreams,
			Usage:   "dump节点上的所有流信息",
		},
		"stream": {
			Handler: parser.dumpStream,
			Usage:   "分析流的详细信息",
		},
		"bw": {
			Handler: parser.CalcTotalBw,
			Usage:   "计算总带宽",
		},
		"stopstream": {
			Handler: parser.stopStream,
			Usage:   "停止的边缘节点所有流",
		},
		"ispchk": {
			Handler: parser.nodeIspChk,
			Usage:   "检查是否有节点存在多个isp的情况",
		},
		"coverchk": {
			Handler: parser.CoverChk,
			Usage:   "分析ck的localAddr和remoteAddr,判断返点是否符合预期",
		},
		"bwdis": {
			Handler: parser.BwDis, // 按省份、运营商，带宽分布
			Usage:   "获取某个bucket跑量在各个省份/大区/isp分布情况",
		},
		"pcdn": {
			Handler: parser.Pcdn,
			Usage:   "请求pcdn接口",
		},
		"dyplay": {
			Handler: parser.DyPlay,
			Usage:   "播放dy地址",
		},
		"stag": {
			Handler: parser.Staging,
			Usage:   "staging",
		},
		"dumproot": {
			Handler: parser.DumpRoots,
			Usage:   "dump root节点详情",
		},
		"dyPcdn": {
			Handler: parser.DyPcdn,
			Usage:   "选择一个有root的大区,且不存在该流的非root节点",
		},
		"nodebyip": {
			Handler: parser.GetNodeByIp,
			Usage:   "通过ip获取节点id",
		},
		"pathquery": {
			Handler: parser.PathqueryReq,
			Usage:   "请求pathquery",
			Depends: []*bool{&conf.Redis, &conf.NodeInfo},
		},
		"area": {
			Handler: parser.Province2Area,
			Usage:   "省份转换为大区",
		},
		"dymetrics": {
			Handler: parser.GetDyMetrics,
			Usage:   "获取dy异常指标",
		},
		"dytimeout": {
			Handler: parser.GetDyTimeout,
			Usage:   "获取dy一天内的topn 节点推送超时率数据",
		},
		"pcdns": {
			Handler: parser.Pcdns,
			Usage:   "遍历province*isp, 请求pcdn",
		},
		"ck": {
			Handler: parser.Ck,
			Usage:   "查询clickhouse",
		},
		"nali": {
			Handler: parser.Nali,
			Usage:   "解析ip的地理位置信息",
		},
		"looppcdn": {
			Handler: parser.LoopPcdn,
			Usage:   "循环请求pcdn",
		},
		"isroot": {
			Handler: parser.IsRoot,
			Usage:   "判断节点是不是root",
			Depends: []*bool{&conf.Redis, &conf.NodeInfo},
		},
		"qpm": {
			Handler: parser.DumpQPM,
			Usage:   "dump qpm 信息",
		},
		"pcdnerrmonitor": {
			Handler: parser.pcdnErrMonitor,
			Usage:   "dump pcdn err信息",
			Depends: []*bool{&conf.Redis, &conf.NodeInfo},
		},
		"aksk": {
			Handler: func() {
				util.GetAkSk(conf)
			},
			Usage: "通过uid获取aksk信息",
		},
		"getdomain": {
			Handler: parser.GetDomain,
			Usage:   "获取domain信息",
		},
		"updatedomain": {
			Handler: parser.UpdateDomain,
			Usage:   "更新domain信息",
		},
		"dyori": {
			Handler: parser.DyOriginal,
			Usage:   "斗鱼回源地址",
		},
		"niulink": {
			Handler: func() {
				util.NiuLink(conf)
			},
			Usage: "niulink 获取动态节点信息",
		},
		"k8s": {
			Handler: parser.K8s,
			Usage:   "k8s获取节点列表",
			Depends: []*bool{&conf.Redis, &conf.NodeInfo},
		},
		"xs": {
			Handler: parser.XsPlay,
			Usage:   "播放dy xs流(go run . -cmd xs -pcdn 123.159.206.211:80 -domain www.dytest.cn -stream 5684726rb3S89L1X_2000p -bkt live -sourceid 1826426132)",
		},
		"gb": {
			Handler: parser.Gb,
			Usage:   "gb",
		},
		"tcpdump": {
			Handler: parser.Tcpdump,
			Usage:   "tcpdump",
		},
		"mocksrv": {
			Handler: mock.MockSrv,
			Usage:   "qvs-server mock",
		},
		"nodedis": {
			Handler: parser.NodeDis,
			Depends: []*bool{&conf.Redis, &conf.NodeInfo, &conf.NeedIpParer},
			Usage:   "节点分布",
		},
		"mockthemisd": {
			Handler: mock.MockThemisd,
			Usage:   "pili-themisd mock",
		},
		"res": {
			Handler: parser.Res,
			Depends: []*bool{&conf.NodeInfo},
			Usage:   "评估带宽资源缺失情况",
		},
		"dumpnodes": {
			Handler: parser.DumpAllNodes,
			Depends: []*bool{&conf.Redis, &conf.NodeInfo},
			Usage:   "dump所有节点信息, 保存json, qup上传",
		},
		"http": {
			Handler: func() {
				util.Http(conf)
			},
			Usage: "http 请求",
		},
		"pushtimeout": {
			Handler: parser.PushTimeout,
			Usage:   "推流超时",
		},
		"gbcli": {
			Handler: parser.GbCli,
			Usage:   "gb camera 模拟器",
		},
		"sipsess": {
			Handler: parser.SipSess,
			Usage:   "sip session 模拟器",
		},
		"talk": {
			Handler: parser.Talk,
			Usage:   "先请求create_audio_channel，然后发送invite talk",
		},
		"invite": {
			Handler: parser.Invite,
			Usage:   "请求gb拉流",
		},
		"kodo": {
			Handler: parser.signResource,
			Usage:   "kodo上的资源签算下载url",
		},
		"nodes": {
			Handler: parser.getQvsStaticNodes,
			Usage:   "获取QVS所有节点信息",
		},
		"redis": {
			Handler: util.Redis,
			Usage:   "dump mik_netprobe_runtime_nodes_map数据，写入到redis",
		},
		"report": {
			Handler: parser.Report,
			Depends: []*bool{&conf.Redis, &conf.NodeInfo, &conf.NeedNodeStreamInfo},
			Usage:   "dump stream report数据",
		},
	}
	if conf.Help {
		dumpCmdMap(cmdMap)
	}
	config.InitConf(cmdMap, conf)
	redisCli := &redis.ClusterClient{}
	if conf.Redis && !conf.Help {
		redisCli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:      conf.RedisAddrs,
			MaxRetries: 3,
			PoolSize:   30,
		})

		err := redisCli.Ping(context.Background()).Err()
		if err != nil {
			log.Fatalf("%+v", err)
		}
	}
	var ipParser *ipdb.City
	if conf.NeedIpParer && !conf.Help {
		//qlog.SetOutputLevel(5)
		var err error
		ipParser, err = ipdb.NewCity(conf.IPDB)
		if err != nil {
			log.Fatalf("[IPDB NewCity] err: %+v\n", err)
		}
	}
	ck := newCk(conf)

	parser.CmdMap = cmdMap
	parser.RedisCli = redisCli
	parser.IpParser = ipParser
	parser.CK = ck

	return parser
}

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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
	//qlog "github.com/qbox/pili/base/qiniu/log.v1"
)

func dumpCmdMap(cmdMap map[string]CmdInfo) {
	for cmd, info := range cmdMap {
		fmt.Printf("%s\n\t%s\n", cmd, info.Usage)
	}
	fmt.Println()
	flag.PrintDefaults()
	os.Exit(0)
}

func newParser(conf *Config) *Parser {

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
	parser := &Parser{
		redisCli: redisCli,
		ipParser: ipParser,
		ck:       ck,
		conf:     conf,
	}

	cmdMap := map[string]CmdInfo{
		"hlschk": {
			Handler: parser.HlsChk,
			Usage:   "hls带宽统计",
		},
		"mockagent": {
			Handler: parser.MockAgent,
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
		"lag": {
			Handler: parser.LagAnalysis,
			Usage:   "分析从ck下载的streamd qos数据, 分析卡顿率高的原因",
		},
		"pcdndbg": {
			Handler: parser.PcdnDbg,
			Usage:   "302返点本省/本大区覆盖率检查,请求sched的playcheck接口",
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
		},
		"area": {
			Handler: parser.Province2Area,
			Usage:   "省份转换为大区",
		},
		"dymetrics": {
			Handler: parser.GetDyMetrics,
			Usage:   "获取dy异常指标",
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
	}
	if conf.Help {
		dumpCmdMap(cmdMap)
	}
	parser.CmdMap = cmdMap
	return parser
}

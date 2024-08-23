package main

import (
	"flag"
	"log"

	qconfig "github.com/qiniu/x/config"
)

const (
	confFile = "./middle-source.json"
)

func loadCfg() *Config {
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	flag.StringVar(&conf.Bucket, "bkt", "douyu", "bucket ID")
	flag.StringVar(&conf.Node, "node", "", "node ID")
	flag.StringVar(&conf.Stream, "stream", "", "stream ID")
	flag.StringVar(&conf.PrometheusAddr, "prometheus", "101.132.36.201:9091", "prometheus addr")
	flag.BoolVar(&conf.CheckNode, "chknode", false, "是否需要检查节点的状态")
	flag.BoolVar(&conf.Monitor, "monitor", false, "node monitor")
	flag.BoolVar(&conf.NeedIpParer, "ipparser", false, "是否需要ip库")
	flag.BoolVar(&conf.NeedCk, "ck", false, "是否需要clickhouse")
	flag.BoolVar(&conf.NeedNodeStreamInfo, "streamNodes", false, "是否需要流所在的节点信息")
	flag.BoolVar(&conf.Bw, "bw", false, "获取总建设带宽+总可用带宽")
	flag.BoolVar(&conf.Streams, "streams", false, "dump所有流信息")
	flag.StringVar(&conf.LagFile, "lagfile", "", "分析streamd上报的卡顿数据的文件")
	flag.BoolVar(&conf.NodeInfo, "nodeinfo", false, "是否需要查询redis获取节点数据")
	flag.BoolVar(&conf.Prometheus, "prometheusEnable", false, "是否需要加载prometheus")
	flag.BoolVar(&conf.Redis, "redis", false, "是否需要加载redis")
	flag.Parse()
	return &conf
}

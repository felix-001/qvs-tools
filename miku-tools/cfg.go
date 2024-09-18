package main

import (
	"flag"
	"log"

	qconfig "github.com/qiniu/x/config"
)

const (
	confFile = "/tmp/middle-source.json"
)

func loadCfg() *Config {
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	flag.StringVar(&conf.Bucket, "bkt", "douyu", "bucket ID")
	flag.StringVar(&conf.Node, "node", "", "node ID")
	flag.StringVar(&conf.Stream, "stream", "288016rlols5_2000p", "stream ID")
	//flag.StringVar(&conf.PrometheusAddr, "prometheus", "101.132.36.201:9091", "prometheus addr")
	//flag.BoolVar(&conf.NeedIpParer, "ipparser", false, "是否需要ip库")
	//flag.BoolVar(&conf.NeedCk, "ck", false, "是否需要clickhouse")
	flag.BoolVar(&conf.NeedNodeStreamInfo, "streamNodes", false, "是否需要流所在的节点信息")
	flag.StringVar(&conf.LagFile, "lagfile", "", "分析streamd上报的卡顿数据的文件")
	//flag.BoolVar(&conf.NodeInfo, "nodeinfo", false, "是否需要查询redis获取节点数据")
	flag.BoolVar(&conf.Prometheus, "prometheusEnable", false, "是否需要加载prometheus")
	//flag.BoolVar(&conf.Redis, "redis", false, "是否需要加载redis")
	flag.StringVar(&conf.DnsResFile, "dnschk", "", "阿里网络拨测工具结果文件")
	flag.StringVar(&conf.PathqueryLogFile, "pathqueryfile", "", "解析elk下载的pathquery日志文件,判断回源路径是否符合预期")
	flag.StringVar(&conf.Ak, "ak", "", "ak")
	flag.StringVar(&conf.Sk, "sk", "", "sk")
	flag.StringVar(&conf.Cmd, "cmd", "streams", "需要执行的子命令(hlschk)")
	flag.StringVar(&conf.QosFile, "qosfile", "", "解析从ck上下载的streamd qos文件，检查locate addr和remote addr是否跨isp、省、大区")
	flag.BoolVar(&conf.Help, "h", false, "help")
	flag.StringVar(&conf.Isp, "isp", "电信", "isp")
	flag.StringVar(&conf.Province, "province", "浙江", "省份")
	flag.StringVar(&conf.Area, "area", "华东", "大区")
	flag.StringVar(&conf.Pcdn, "pcdn", "", "指定pcdn的ip:port")
	flag.StringVar(&conf.Ip, "ip", "", "通过ip获取node id")
	flag.Parse()

	if conf.Cmd == "ispchk" {
		conf.Redis = true
		conf.NeedIpParer = true
		conf.NodeInfo = true
	}

	if conf.Cmd == "hlschk" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedNodeStreamInfo = true
	}
	if conf.Node != "" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedNodeStreamInfo = true
	}
	if conf.Cmd == "streams" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedNodeStreamInfo = true
		conf.NeedIpParer = true
	}
	if conf.Cmd == "bwdis" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedNodeStreamInfo = true
		conf.NeedIpParer = true
	}
	if conf.Cmd == "coverchk" {
		conf.NeedIpParer = true
	}
	if conf.Cmd == "dumproot" {
		conf.Redis = true
		conf.NodeInfo = true
	}
	if conf.Cmd == "nodebyip" {
		conf.Redis = true
		conf.NodeInfo = true
	}
	return &conf
}

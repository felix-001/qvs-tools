package main

import (
	"flag"
	"log"
	"time"

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

	t := time.Now().Format("2006-01-02 15:04:05")

	flag.StringVar(&conf.Bucket, "bkt", "douyu", "bucket ID")
	flag.StringVar(&conf.Node, "node", "2b8f0c5a-85d0-3c4a-bbd8-ac77a82d607b-rtc-gdfsh-dls-1-7", "node ID")
	flag.StringVar(&conf.Stream, "stream", "288016rlols5_2000p", "stream ID")
	flag.BoolVar(&conf.NeedNodeStreamInfo, "streamNodes", false, "是否需要流所在的节点信息")
	flag.StringVar(&conf.LagFile, "lagfile", "", "分析streamd上报的卡顿数据的文件")
	flag.BoolVar(&conf.Prometheus, "prometheusEnable", false, "是否需要加载prometheus")
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
	flag.StringVar(&conf.Ip, "ip", "221.227.254.220", "通过ip获取node id")
	flag.StringVar(&conf.T, "t", t, "时间, 格式: 2006-01-02 15:04:05")
	flag.StringVar(&conf.Query, "q", "", "查询ck的语句")
	flag.IntVar(&conf.N, "n", 70, "pcdn循环请求次数")
	flag.StringVar(&conf.QpmFile, "qpmfile", "", "dump出来的内存中的qpm数据")
	flag.StringVar(&conf.PcdnErr, "pcdnerr", "", "请求pcdn是附带的pcdn err节点id")
	flag.StringVar(&conf.User, "user", "volcengine", "请求playcheck 302的user")
	flag.StringVar(&conf.ConnId, "connid", "testConnId", "请求playcheck 302的connId")
	flag.BoolVar(&conf.Https, "https", false, "是否启用https")
	flag.IntVar(&conf.Interval, "interval", 15, "时间间隔")
	flag.StringVar(&conf.SchedIp, "schedip", "10.34.146.62", "调度服务ip")
	flag.StringVar(&conf.Uid, "uid", "", "uid")
	flag.StringVar(&conf.Domain, "domain", "www.voltest2.com", "domain")
	flag.StringVar(&conf.SubCmd, "subcmd", "", "staging subcmd")
	flag.StringVar(&conf.Format, "format", "slice", "playcheck请求原始url的format, 例如: flv m3u8 xs slice")
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
	if conf.Cmd == "pcdns" {
		conf.NeedIpParer = true
	}
	if conf.Cmd == "nali" {
		conf.NeedIpParer = true
	}
	if conf.Cmd == "dytimeout" {
		conf.Redis = true
		conf.NodeInfo = true
	}
	if conf.Cmd == "stag" && conf.SubCmd == "ipv6" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedIpParer = true
	}
	return &conf
}

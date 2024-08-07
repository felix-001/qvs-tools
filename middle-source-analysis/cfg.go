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
	flag.Parse()
	return &conf
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const (
	conf = "/usr/local/etc/qvsdbg.json"
)

type Config struct {
	AdminAddr      string `json:"admin_addr"`
	Start          string `json:"start"`
	End            string `json:"end"`
	StreamId       string `json:"streamId"`
	StreamPullFail bool
	Re             string
	Verbose        bool
	PullStream     bool
	HttpSrv        bool
	Sip            bool
	WritePyToNode  bool
	SearchThemisd  bool
}

func checkConf(config *Config) error {
	if config.AdminAddr == "" {
		return fmt.Errorf("admin ip empty")
	}
	/*
		if config.GbId == "" {
			return fmt.Errorf("gbid empty")
		}
	*/
	return nil
}

func parseConsole(config *Config) {
	currentTime := time.Now()
	end := currentTime.Format("2006-01-02 15:04:05")
	start := currentTime.Add(-time.Hour).Format("2006-01-02 15:04:05")

	flag.StringVar(&config.AdminAddr, "addr", "10.20.76.42:7277", "admin addr")
	flag.StringVar(&config.StreamId, "sid", "", "stream id")
	flag.StringVar(&config.Re, "re", "", "捞日志的正则表达式")
	flag.StringVar(&config.Start, "start", start, "开始时间,格式为2023-11-05 19:20:00")
	flag.StringVar(&config.End, "end", end, "结束时间,格式为2023-11-05 19:20:00")
	flag.BoolVar(&config.StreamPullFail, "s", false, "拉流失败获取日志")
	flag.BoolVar(&config.Verbose, "v", false, "是否打印更详细的日志")
	flag.BoolVar(&config.PullStream, "pull", false, "捞取拉流日志")
	flag.BoolVar(&config.HttpSrv, "srv", false, "是否启动http server")
	flag.BoolVar(&config.Sip, "sip", false, "是否捞取sip日志")
	flag.BoolVar(&config.WritePyToNode, "w", false, "是否需要把py脚本写入到节点")
	flag.BoolVar(&config.SearchThemisd, "themisd", false, "是否搜索themisd日志")

	flag.Parse()
}

func loadConf(config *Config) error {
	parseConsole(config)
	if _, err := os.Stat(conf); err == nil {
		b, err := ioutil.ReadFile(conf)
		if err != nil {
			return fmt.Errorf("%s not found", conf)
		}
		if err := json.Unmarshal(b, config); err != nil {
			return fmt.Errorf("parse conf err: %v", err)
		}
	}
	if err := checkConf(config); err != nil {
		flag.PrintDefaults()
		return err
	}
	return nil
}

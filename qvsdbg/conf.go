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
	AdminAddr string `json:"admin_addr"`
	GbId      string `json:"gbid"`
	ChId      string `json:"chid"`
	Start     string `json:"start"`
	End       string `json:"end"`
	PdrToken  string `json:"pdr_token"`
}

func checkConf(config *Config) error {
	if config.AdminAddr == "" {
		return fmt.Errorf("admin ip empty")
	}
	if config.GbId == "" {
		return fmt.Errorf("gbid empty")
	}
	return nil
}

func parseConsole(config *Config) {
	currentTime := time.Now()
	end := currentTime.Format("2006-01-02 15:04:05")
	start := currentTime.Add(-time.Hour).Format("2006-01-02 15:04:05")
	flag.StringVar(&config.AdminAddr, "addr", "10.20.76.42:7277", "admin addr")
	flag.StringVar(&config.GbId, "gbid", "", "gbid")
	flag.StringVar(&config.ChId, "chid", "", "chid")
	flag.StringVar(&config.PdrToken, "token", "", "pdr token")

	flag.StringVar(&config.Start, "start", start, "开始时间,格式为2023-11-05 19:20:00")
	flag.StringVar(&config.End, "end", end, "结束时间,格式为2023-11-05 19:20:00")
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

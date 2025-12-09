package qos

import (
	"mikutool/public/util"

	"github.com/qbox/pili/common/ipdb.v1"
)

// AggregatedData 聚合数据结构
type AggregatedData struct {
	IP         string   `json:"ip"`
	TotalCount int      `json:"total_count"`
	LagCount   int      `json:"lag_count"`
	Prov       string   `json:"prov"`
	Isp        string   `json:"isp"`
	RemoteIps  []string `json:"remote_ips"`
	LagRate    float64  `json:"lag_rate"`
}

type CdnAggregateData struct {
	IP             string   `json:"ip"`
	TotalUserCount int      `json:"total_user_count"`
	LagIps         []string `json:"lag_ips"`
}

type AggData struct {
	CountryCntMap map[string]int
	AreaCntMap    map[string]int
	ProvCntMap    map[string]int
	AreaLagCntMap map[string]int
	ProvLagCntMap map[string]int
}

// MinuteAggregatedData 按分钟聚合数据结构
type MinuteAggregatedData struct {
	Timestamp    string               `json:"timestamp"`
	Percent      float64              `json:"percent"`
	LagCount     int                  `json:"lag_count"`
	TotalCount   int                  `json:"total_count"`
	LagUserCnt   int                  `json:"lag_user_cnt"`
	TotalUserCnt int                  `json:"total_user_cnt"`
	LagCdnCnt    int                  `json:"lag_cdn_cnt"`
	TotalCdnCnt  int                  `json:"total_cdn_cnt"`
	Reports      []util.QualityReport `json:"report"`
}

// QOSRequest 前端查询请求结构
type QOSRequest struct {
	AppName     string     `json:"appName"`
	StartTime   string     `json:"startTime"`
	EndTime     string     `json:"endTime"`
	StreamID    string     `json:"streamId"`
	Domain      string     `json:"domain"`
	UID         string     `json:"uid"`
	UserIp      string     `json:"userIp"`
	CdnIp       string     `json:"cdnIp"`
	Hour        string     `json:"hour"`
	FuzzySearch bool       `json:"fuzzySearch"`
	LogLevel    string     `json:"logLevel"`
	RawData     bool       `json:"rawData"`
	IpParser    *ipdb.City `json:"ipParser"`
}

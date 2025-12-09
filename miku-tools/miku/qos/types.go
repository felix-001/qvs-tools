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
	AppName     string       `json:"appName"`
	StartTime   string       `json:"startTime"`
	EndTime     string       `json:"endTime"`
	StreamID    string       `json:"streamId"`
	Domain      string       `json:"domain"`
	UID         string       `json:"uid"`
	UserIp      string       `json:"userIp"`
	CdnIp       string       `json:"cdnIp"`
	Hour        string       `json:"hour"`
	FuzzySearch bool         `json:"fuzzySearch"`
	LogLevel    string       `json:"logLevel"`
	RawData     bool         `json:"rawData"`
	Charts      ChartOptions `json:"charts"`
	IpParser    *ipdb.City   `json:"ipParser"`
}

// ChartOptions 图表展示选项
type ChartOptions struct {
	CdnIpQuality         bool `json:"cdnIpQuality"`
	CdnLagUsers          bool `json:"cdnLagUsers"`
	CdnLagRate           bool `json:"cdnLagRate"`
	LagUserRatio         bool `json:"lagUserRatio"`
	ClientIpQuality      bool `json:"clientIpQuality"`
	LagRateTrend         bool `json:"lagRateTrend"`
	NodeLagRatio         bool `json:"nodeLagRatio"`
	RetryRate            bool `json:"retryRate"`
	RetryCount           bool `json:"retryCount"`
	CountryDistribution  bool `json:"countryDistribution"`
	RegionDistribution   bool `json:"regionDistribution"`
	ProvinceDistribution bool `json:"provinceDistribution"`
	RegionDelay          bool `json:"regionDelay"`
	ProvinceDelay        bool `json:"provinceDelay"`
	StreamDelay          bool `json:"streamDelay"`
	InternalRetryLag     bool `json:"internalRetryLag"`
	ClientRetryLag       bool `json:"clientRetryLag"`
	OnlineUsers          bool `json:"onlineUsers"`
	OnlineStreams        bool `json:"onlineStreams"`
	VideoFps             bool `json:"videoFps"`
	AudioFps             bool `json:"audioFps"`
}

// OnlineUserAggregatedData 在线用户聚合数据结构
type OnlineUserAggregatedData struct {
	Timestamp string `json:"timestamp"`
	OnlineNum int    `json:"online_num"`
}

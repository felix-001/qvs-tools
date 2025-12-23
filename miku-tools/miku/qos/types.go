package qos

import (
	"mikutool/public/util"

	"github.com/qbox/pili/common/ipdb.v1"
)

// AggregatedData 聚合数据结构
type AggregatedData struct {
	IP            string   `json:"ip"`
	TotalCount    int      `json:"total_count"`    // 总样本数
	LagCount      int      `json:"lag_count"`      // 卡顿样本数
	LagRate       float64  `json:"lag_rate"`       // 本节点卡顿样本占卡顿样本总数的比例
	Prov          string   `json:"prov"`           // 节点省份
	Isp           string   `json:"isp"`            // 节点运营商
	NormalIps     []string `json:"normal_ips"`     // 不卡顿的ip列表
	LagUsrCnt     int      `json:"lag_usr_cnt"`    // 卡顿用户个数
	TotalUsrCnt   int      `json:"total_usr_cnt"`  // 总用户数
	LagUsrRate    float64  `json:"lag_usr_rate"`   // 卡顿用户比
	LagIps        []string `json:"lag_ips"`        // 卡顿的ip列表
	NormalStreams []string `json:"normal_streams"` // 流列表
	LagStreams    []string `json:"lag_streams"`    // 卡顿的流列表
}

type UserAggregatedData struct {
	ClientIP      string   `json:"client_ip"`
	TotalCount    int      `json:"total_count"`    // 总样本数
	LagCount      int      `json:"lag_count"`      // 卡顿样本数
	Percent       float64  `json:"percent"`        // 卡顿率
	Weight        float64  `json:"weight"`         // 本用户卡顿样本占卡顿样本总数的比例
	Prov          string   `json:"prov"`           // 用户省份
	Isp           string   `json:"isp"`            // 用户运营商
	NormalIps     []string `json:"normal_ips"`     // 不卡顿的节点ip列表
	LagNodeCnt    int      `json:"lag_node_cnt"`   // 卡顿节点个数
	TotalNodeCnt  int      `json:"total_node_cnt"` // 总用户数
	LagNodeRate   float64  `json:"lag_node_rate"`  // 卡顿节点比
	LagNodeIps    []string `json:"lag_node_ips"`   // 卡顿的节点ip列表
	NormalStreams []string `json:"normal_streams"` // 流列表
	LagStreams    []string `json:"lag_streams"`    // 卡顿的流列表
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
	AppName     string `json:"appName"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	StreamID    string `json:"streamId"`
	Domain      string `json:"domain"`
	UID         string `json:"uid"`
	UserIp      string `json:"userIp"`
	CdnIp       string `json:"cdnIp"`
	Hour        string `json:"hour"`
	FuzzySearch bool   `json:"fuzzySearch"`
	LogLevel    string `json:"logLevel"`
	RawData     bool   `json:"rawData"`
	//Charts      ChartOptions `json:"charts"`
	IpParser *ipdb.City `json:"ipParser"`
	Chart    string     `json:"chart"`
	Protocol string     `json:"protocol"`
}

// ChartOptions 图表展示选项
type ChartOptions struct {
	CustomerUpstreamLagRate    bool `json:"customerUpstreamLagRate"`    // 回客户源站百秒卡顿率趋势图
	InternalUpstreamLagRate    bool `json:"internalUpstreamLagRate"`    // 内部回源百秒卡顿率趋势图
	InternalUpstreamRetryTimes bool `json:"internalUpstreamRetryTimes"` // 内部回源重试次数趋势图
	InternalUpstreamRetryRate  bool `json:"internalUpstreamRetryRate"`  // 内部回源重试率趋势图
	CdnIpQuality               bool `json:"cdnIpQuality"`
	CdnLagUsers                bool `json:"cdnLagUsers"`
	CdnLagRate                 bool `json:"cdnLagRate"`
	LagUserRatio               bool `json:"lagUserRatio"`
	ClientIpQuality            bool `json:"clientIpQuality"`
	LagRateTrend               bool `json:"lagRateTrend"`
	CountryDistribution        bool `json:"countryDistribution"`
	RegionDistribution         bool `json:"regionDistribution"`
	ProvinceDistribution       bool `json:"provinceDistribution"`
	RegionDelay                bool `json:"regionDelay"`
	ProvinceDelay              bool `json:"provinceDelay"`
	StreamDelay                bool `json:"streamDelay"`
	OnlineUsers                bool `json:"onlineUsers"`
	OnlineStreams              bool `json:"onlineStreams"`
	VideoFps                   bool `json:"videoFps"`
	AudioFps                   bool `json:"audioFps"`
}

// OnlineUserAggregatedData 在线用户聚合数据结构
type OnlineUserAggregatedData struct {
	Timestamp string `json:"timestamp"`
	OnlineNum int    `json:"online_num"`
}

type LineChartData struct {
	XAxis       []string `json:"xAxis"`
	YAxis       []string `json:"yAxis"`
	Title       string   `json:"title"`
	Color       string   `json:"color"`
	SeriesTitle string   `json:"seriesTitle"`
	XType       string   `json:"xType"` // 定义 X 轴的数据类型, category 为分类数据，time 为时间型, value为数值数据
}
type PieDataItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type PieChartData struct {
	Title string        `json:"title"`
	Data  []PieDataItem `json:"data"`
}

type ChartInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ChartData struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

const (
	ChartTypeLine  = "line"
	ChartTypePie   = "pie"
	ChartTypeTable = "table"
)

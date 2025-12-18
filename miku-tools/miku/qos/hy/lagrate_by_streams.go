package hy

import (
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

// 卡顿率按流分布

func init() {
	qos.RegisterChartGenerator("hyLagrateByStreams", &LagRateByStreams{})
}

type LagRateByStreams struct {
}

func (l *LagRateByStreams) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" {
		return ""
	}
	sql := qos.BuildHyLagRateByStreamsSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyLagRateByStreamsReports []util.HyLagRateByStreamsReport
	if err := util.TrinoQuery("miku", sql, &hyLagRateByStreamsReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return ""
	}
	return hyLagRateByStreamsReports
	//return qos.GenerateLagRateByStreamsTable(hyLagRateByStreamsReports)
}

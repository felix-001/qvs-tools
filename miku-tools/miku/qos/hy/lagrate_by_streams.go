package hy

import (
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

// 卡顿率按流分布

func init() {
	qos.RegisterChartGenerator(&LagRateByStreams{})
}

type LagRateByStreams struct {
}

func (l *LagRateByStreams) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" && req.AppName != "huyap2p" {
		return qos.ChartData{Data: "", Type: "error"}
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
	log.Println("len hyLagRateByStreamsReports", len(hyLagRateByStreamsReports))
	var lagTotal int
	for _, report := range hyLagRateByStreamsReports {
		lagTotal += *report.LagCnt
	}
	for i, report := range hyLagRateByStreamsReports {
		if report.LagCnt == nil {
			continue
		}
		weight := float64(*report.LagCnt) * 100.0 / float64(lagTotal)
		hyLagRateByStreamsReports[i].Weight = &weight
	}
	return qos.ChartData{Data: hyLagRateByStreamsReports, Type: qos.ChartTypeTable}
}

func (l *LagRateByStreams) ID() string {
	return "chart_hyLagrateByStreams"
}

func (l *LagRateByStreams) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    l.ID(),
		Title: "虎牙按流卡顿率表格",
	}
}

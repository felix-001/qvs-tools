package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&LagNodeRate{})
}

// 卡顿的节点数占比

type LagNodeRate struct {
}

func (l *LagNodeRate) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.LagNodeRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.LagNodeRate)
		}
	}
	chartData := qos.GenerateLineChartData("卡顿节点占总节点数的百分比", "占比", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
	//return qos.GenerateCdnLagRatioChartHTML(hyLagRateReports)
}

func (l *LagNodeRate) ID() string {
	return "chart_lagNodesPercent"
}

func (l *LagNodeRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    l.ID(),
		Title: "卡顿节点占总节点数的百分比",
	}
}

package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator("lagNodesPercent", &LagNodeRate{})
}

// 卡顿的节点数占比

type LagNodeRate struct {
}

func (l *LagNodeRate) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
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
	return chartData
	//return qos.GenerateCdnLagRatioChartHTML(hyLagRateReports)
}

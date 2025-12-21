package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 卡顿用户占总用户数的百分比

func init() {
	qos.RegisterChartGenerator(&LagUsrRate{})
}

type LagUsrRate struct {
}

func (l *LagUsrRate) ID() string {
	return "chart_hyLagUsrPercent"
}

func (l *LagUsrRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    l.ID(),
		Title: "卡顿用户占总用户数的百分比",
	}
}

func (l *LagUsrRate) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.LagUsrRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.LagUsrRate)
		}
	}
	chartData := qos.GenerateLineChartData("卡顿用户占总用户数的百分比", "占比", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

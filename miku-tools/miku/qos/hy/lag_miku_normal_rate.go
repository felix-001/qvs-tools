package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&LagMikuNormalRate{})
}

// 虎牙推流MIKU非转码流卡顿率

type LagMikuNormalRate struct {
}

func (l *LagMikuNormalRate) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.MikuNormalLagRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.MikuNormalLagRate)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙推流MIKU非转码流卡顿率", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (l *LagMikuNormalRate) ID() string {
	return "chart_MikuNormalLagRate"
}

func (l *LagMikuNormalRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    l.ID(),
		Title: "虎牙推流MIKU非转码流卡顿率",
	}
}

package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 卡顿用户占总用户数的百分比

func init() {
	qos.RegisterChartGenerator("hyLagUsrPercent", &LagUsrRate{})
}

type LagUsrRate struct {
}

func (l *LagUsrRate) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
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
	return chartData
}

package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 虎牙推流MIKU非转码流占总样本数的比例
func init() {
	qos.RegisterChartGenerator(&HuyaNormalRate{})
}

type HuyaNormalRate struct {
}

func (h *HuyaNormalRate) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.NormalRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.NormalRate)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙推流MIKU非转码流占总样本数的比例", "占比", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaNormalRate) ID() string {
	return "chart_HyNormalRateTrend"
}

func (h *HuyaNormalRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙推流MIKU非转码流占总样本数的比例",
	}
}

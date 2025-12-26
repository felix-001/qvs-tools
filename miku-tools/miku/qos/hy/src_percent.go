package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 虎牙回源整体样本数占总样本数的比例
func init() {
	qos.RegisterChartGenerator(&HuyaSrcRate{})
}

type HuyaSrcRate struct {
}

func (h *HuyaSrcRate) Generate(req qos.QOSRequest) any {
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
			if report.SrcRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.SrcRate)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙回客户源整体样本数占总样本数的比例", "比例", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaSrcRate) ID() string {
	return "chart_HySrcRateTrend"
}

func (h *HuyaSrcRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙回客户源整体样本数占总样本数的比例",
	}
}

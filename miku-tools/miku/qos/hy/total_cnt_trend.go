package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&HuyaTotalCntTrend{})
}

// 虎牙总样本数趋势图

type HuyaTotalCntTrend struct {
}

func (h *HuyaTotalCntTrend) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" && req.AppName != "huyap2p" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.Total == nil {
				continue
			}
			cb(*report.Ts_m, *report.Total)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙总样本数数趋势图", "总样本", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaTotalCntTrend) ID() string {
	return "chart_hyTotalCntTrend"
}

func (h *HuyaTotalCntTrend) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙总样本数数趋势图",
	}
}

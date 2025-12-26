package hy

import (
	"fmt"
	"mikutool/miku/qos"
)

// 虎牙回源流整体卡顿率趋势图
func init() {
	qos.RegisterChartGenerator(&HuyaSrcLagRate{})
}

type HuyaSrcLagRate struct {
}

func (h *HuyaSrcLagRate) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.SrcLagRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.SrcLagRate)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙回源流整体每分钟卡顿率趋势图", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaSrcLagRate) ID() string {
	return "chart_HySrcLagRateTrend"
}

func (h *HuyaSrcLagRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙回源流整体卡顿率趋势图",
	}
}

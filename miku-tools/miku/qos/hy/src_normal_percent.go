package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 虎牙回客户源站非转码流整体样本数占总样本数的比例
func init() {
	qos.RegisterChartGenerator(&HuyaSrcNormalRate{})
}

type HuyaSrcNormalRate struct {
}

func (h *HuyaSrcNormalRate) Generate(req qos.QOSRequest) any {
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
			if report.SrcNormalPercent == nil {
				continue
			}
			cb(*report.Ts_m, *report.SrcNormalPercent)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙回客户源站非转码流整体样本数占总样本数的比例", "比例", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaSrcNormalRate) ID() string {
	return "chart_HySrcNoramalPercent"
}

func (h *HuyaSrcNormalRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙回客户源站非转码流整体样本数占总样本数的比例",
	}
}

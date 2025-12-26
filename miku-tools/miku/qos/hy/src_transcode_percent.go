package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 虎牙回客户源站转码流样本数占总样本数的比例
func init() {
	qos.RegisterChartGenerator(&HuyaSrcTranscodeRate{})
}

type HuyaSrcTranscodeRate struct {
}

func (h *HuyaSrcTranscodeRate) Generate(req qos.QOSRequest) any {
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
			if report.SrcTranscodePercent == nil {
				continue
			}
			cb(*report.Ts_m, *report.SrcTranscodePercent)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙回客户源站转码流样本数占总样本数的比例", "比例", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaSrcTranscodeRate) ID() string {
	return "chart_HySrcTranscodePercent"
}

func (h *HuyaSrcTranscodeRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙回客户源站转码流样本数占总样本数的比例",
	}
}

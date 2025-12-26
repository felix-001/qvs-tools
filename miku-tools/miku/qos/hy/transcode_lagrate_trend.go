package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 虎牙推流MIKU转码流每分钟卡顿率趋势图
func init() {
	qos.RegisterChartGenerator(&HuyaTranscodeLagRate{})
}

type HuyaTranscodeLagRate struct {
}

func (h *HuyaTranscodeLagRate) Generate(req qos.QOSRequest) any {
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
			if report.TrancodeLagRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.TrancodeLagRate)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙推流MIKU转码流每分钟卡顿率趋势图", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaTranscodeLagRate) ID() string {
	return "chart_transcodeHyLagRateTrend"
}

func (h *HuyaTranscodeLagRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙推流MIKU转码流卡顿率趋势图",
	}
}

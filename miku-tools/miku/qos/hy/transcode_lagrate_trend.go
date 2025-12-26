package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"strings"
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
	data := qos.LineChartData{
		Title:       "虎牙推流MIKU转码流每分钟卡顿率趋势图",
		SeriesTitle: "卡顿率",
		Color:       "#1890ff",
		XType:       "category",
		XAxis:       []string{},
		YAxis:       []string{},
	}
	for _, report := range hyLagRateReports {
		if report.Percent == nil {
			continue
		}
		ts := strings.ReplaceAll(*report.Ts_m, "+08:00", "")
		// Remove year from timestamp (format: 2025-12-16T12:49:00)
		if len(ts) >= 10 {
			ts = ts[5:] // Keep everything after the year
		}
		data.XAxis = append(data.XAxis, ts)
		data.YAxis = append(data.YAxis, fmt.Sprintf("%.2f", *report.TrancodeLagRate))
	}
	return qos.ChartData{Data: data, Type: qos.ChartTypeLine}
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

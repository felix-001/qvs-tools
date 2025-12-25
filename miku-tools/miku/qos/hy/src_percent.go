package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"strings"
)

// 虎牙回源样本数占总样本数的比例
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
	data := qos.LineChartData{
		Title:       "虎牙回客户源样本数占总样本数的比例",
		SeriesTitle: "比例",
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
		data.YAxis = append(data.YAxis, fmt.Sprintf("%.2f", *report.SrcRate))
	}
	return qos.ChartData{Data: data, Type: qos.ChartTypeLine}
}

func (h *HuyaSrcRate) ID() string {
	return "chart_HySrcRateTrend"
}

func (h *HuyaSrcRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙回客户源样本数占总样本数的比例",
	}
}

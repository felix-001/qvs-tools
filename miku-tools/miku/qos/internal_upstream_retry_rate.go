package qos

import (
	"fmt"
	"log"
	"strings"
)

// 内部回源重试率趋势图

func init() {
	RegisterChartGenerator("internalUpstreamRetryRate", &InternalUpstreamRetryRate{})
}

type InternalUpstreamRetryRate struct {
}

func (i *InternalUpstreamRetryRate) Generate(req QOSRequest) any {
	streamdReports, err := GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	data := LineChartData{
		Title:       "内部回源重试率",
		SeriesTitle: "重试率",
		Color:       "#1890ff",
		XType:       "category",
		XAxis:       []string{},
		YAxis:       []string{},
	}
	for _, report := range streamdReports {
		if report.Retry_ratio_puller == nil {
			continue
		}
		ts := strings.ReplaceAll(*report.Ts_m, "+08:00", "")
		// Remove year from timestamp (format: 2025-12-16T12:49:00)
		if len(ts) >= 10 {
			ts = ts[5:] // Keep everything after the year
		}
		data.XAxis = append(data.XAxis, ts)
		data.YAxis = append(data.YAxis, fmt.Sprintf("%.2f", *report.Retry_ratio_puller))
	}
	return data
}

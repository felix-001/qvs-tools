package upstream

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"strings"
)

// 回客户源站白秒卡顿率

func init() {
	qos.RegisterChartGenerator("customerUpstreamLagRate", &UsrUpstreamLagRate{})
}

type UsrUpstreamLagRate struct {
}

func (m *UsrUpstreamLagRate) Generate(req qos.QOSRequest) any {
	log.Println("Generating customer upstream lag rate chart")
	streamdReports, err := qos.GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	log.Println("查询结果streamdReports:", len(streamdReports))
	data := qos.LineChartData{
		Title:       "回客户源站百秒卡顿率",
		SeriesTitle: "卡顿率",
		Color:       "#1890ff",
		XType:       "category",
		XAxis:       []string{},
		YAxis:       []string{},
	}
	for _, report := range streamdReports {
		if report.Ratio_lag_puller == nil {
			continue
		}
		ts := strings.ReplaceAll(*report.Ts_m, "+08:00", "")
		// Remove year from timestamp (format: 2025-12-16T12:49:00)
		if len(ts) >= 10 {
			ts = ts[5:] // Keep everything after the year
		}

		data.XAxis = append(data.XAxis, ts)
		data.YAxis = append(data.YAxis, fmt.Sprintf("%.2f", *report.Ratio_lag_puller))
	}
	return data
}

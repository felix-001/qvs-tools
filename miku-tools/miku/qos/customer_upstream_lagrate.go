package qos

import (
	"fmt"
	"log"
	"strings"
)

// 回客户源站白秒卡顿率

func init() {
	RegisterChartGenerator("customerUpstreamLagRate", &UsrUpstreamLagRate{})
}

type UsrUpstreamLagRate struct {
}

func (m *UsrUpstreamLagRate) Generate(req QOSRequest) any {
	if !req.Charts.CustomerUpstreamLagRate {
		return ""
	}
	log.Println("Generating customer upstream lag rate chart")
	streamdReports, err := GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	log.Println("查询结果streamdReports:", len(streamdReports))
	data := LineChartData{
		Title:       "回客户源站百秒卡顿率",
		SeriesTitle: "卡顿率",
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
	html := fmt.Sprintf(`
	<div style="margin-top: 20px;">
		<div style="margin-bottom: 30px;">
			<h4>回客户源站百秒卡顿率</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
	</div>`,
		generateLineChartHTML(streamdReports, "Ts_m", "回客户源站百秒卡顿率", "回客户源站百秒卡顿率"),
	)
	return html
}

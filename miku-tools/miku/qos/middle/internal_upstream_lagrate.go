package middle

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"strings"
)

// 内部回源百秒卡顿率趋势图

func init() {
	qos.RegisterChartGenerator(&InternalUpstreamLagRate{})
}

type InternalUpstreamLagRate struct {
}

func (i *InternalUpstreamLagRate) ID() string {
	return "chart_internalUpstreamLagRate"
}

func (i *InternalUpstreamLagRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    i.ID(),
		Title: "内部回源百秒卡顿率",
	}
}

func (i *InternalUpstreamLagRate) Generate(req qos.QOSRequest) any {
	streamdReports, err := qos.GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	data := qos.LineChartData{
		Title:       "内部回源百秒卡顿率",
		SeriesTitle: "卡顿率",
		Color:       "#1890ff",
		XType:       "category",
		XAxis:       []string{},
		YAxis:       []string{},
	}
	for _, report := range streamdReports {
		if report.Ratio_lag_internal_player == nil {
			continue
		}
		ts := strings.ReplaceAll(*report.Ts_m, "+08:00", "")
		// Remove year from timestamp (format: 2025-12-16T12:49:00)
		if len(ts) >= 10 {
			ts = ts[5:] // Keep everything after the year
		}
		data.XAxis = append(data.XAxis, ts)
		data.YAxis = append(data.YAxis, fmt.Sprintf("%.2f", *report.Ratio_lag_internal_player))
	}
	return data
}

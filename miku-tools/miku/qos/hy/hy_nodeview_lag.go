package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"

	"github.com/go-echarts/go-echarts/v2/opts"
)

func init() {
	qos.RegisterChartGenerator("hy_nodeview_lag", &HyNodeViewLag{})
}

type HyNodeViewLag struct {
}

func (h *HyNodeViewLag) Generate(req qos.QOSRequest) string {
	if req.CdnIp == "" {
		return ""
	}
	sql := qos.BuildHyCdnIpLagSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyCdnIpLagReports []util.HyLagReport
	if err := util.TrinoQuery("miku", sql, &hyCdnIpLagReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyCdnIpLagReports:", len(hyCdnIpLagReports))
	// 使用通用的折线图生成函数来显示CDN IP卡顿数据
	var xAxisData []string
	var yAxisData []opts.LineData

	for _, report := range hyCdnIpLagReports {
		if report.Ts_m != nil && report.Percent != nil {
			xAxisData = append(xAxisData, *report.Ts_m)
			yAxisData = append(yAxisData, opts.LineData{
				Value: *report.Percent,
			})
		}
	}

	html := qos.GenerateLineChart("CDN IP卡顿率", "CDN IP卡顿率", "", xAxisData, yAxisData)
	return html
}

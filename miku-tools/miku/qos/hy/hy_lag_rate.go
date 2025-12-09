package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

func init() {
	qos.RegisterChartGenerator("hy_lag_rate", &HuyaLagRate{})
}

type HuyaLagRate struct {
}

func (h *HuyaLagRate) Generate(req qos.QOSRequest) string {
	sql := qos.BuildHyLagRateSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyLagRateReports []util.HyLagReport
	if err := util.TrinoQuery("miku", sql, &hyLagRateReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	html := qos.GenerateMinuteLagRateChartHTML(hyLagRateReports)

	return html
}

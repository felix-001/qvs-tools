package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator("hy_lag_rate", &HuyaLagRate{})
}

// 每分钟卡顿率趋势图

type HuyaLagRate struct {
}

func (h *HuyaLagRate) Generate(req qos.QOSRequest) string {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	html := qos.GenerateMinuteLagRateChartHTML(hyLagRateReports)

	return html
}

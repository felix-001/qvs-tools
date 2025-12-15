package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator("lag_node_rate", &LagNodeRate{})
}

// 卡顿的节点数占比

type LagNodeRate struct {
}

func (l *LagNodeRate) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	return qos.GenerateCdnLagRatioChartHTML(hyLagRateReports)
}

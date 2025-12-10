package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 卡顿用户占总用户数的百分比

func init() {
	qos.RegisterChartGenerator("lag_usr_rate", &LagUsrRate{})
}

type LagUsrRate struct {
}

func (l *LagUsrRate) Generate(req qos.QOSRequest) string {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	return qos.GenerateLagUserRatioChartHTML(hyLagRateReports)
}

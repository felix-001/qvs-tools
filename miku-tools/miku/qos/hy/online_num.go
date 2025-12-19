package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 在线用户数折线图

func init() {
	qos.RegisterChartGenerator("onlineUsers", &OnlineNum{})
}

type OnlineNum struct {
}

func (o *OnlineNum) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.TotalUsrCnt == nil {
				continue
			}
			cb(*report.Ts_m, *report.TotalUsrCnt)
		}
	}
	chartData := qos.GenerateLineChartData("在线用户数", "人数", fn)
	return chartData
}

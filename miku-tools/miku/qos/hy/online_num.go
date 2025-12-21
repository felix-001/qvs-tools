package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 在线用户数折线图

func init() {
	qos.RegisterChartGenerator(&OnlineNum{})
}

type OnlineNum struct {
}

func (o *OnlineNum) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
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
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (o *OnlineNum) ID() string {
	return "chart_onlineUsers"
}

func (o *OnlineNum) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    o.ID(),
		Title: "每分钟在线用户数趋势图",
	}
}

package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 在线用户数折线图

func init() {
	qos.RegisterChartGenerator("online_num", &OnlineNum{})
}

type OnlineNum struct {
}

func (o *OnlineNum) Generate(req qos.QOSRequest) string {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	return qos.GenerateOnlineUsersChartHTML(hyLagRateReports)
}

package hy

import "mikutool/miku/qos"

// 在线用户数折线图

func init() {
	qos.RegisterChartGenerator("online_num", &OnlineNum{})
}

type OnlineNum struct {
}

func (o *OnlineNum) Generate(req qos.QOSRequest) string {
	return ""
}

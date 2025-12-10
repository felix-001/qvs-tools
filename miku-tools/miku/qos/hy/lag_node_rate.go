package hy

import "mikutool/miku/qos"

func init() {
	qos.RegisterChartGenerator("lag_node_rate", &LagNodeRate{})
}

// 卡顿的节点数占比

type LagNodeRate struct {
}

func (l *LagNodeRate) Generate(req qos.QOSRequest) string {
	return ""
}

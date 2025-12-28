package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&HuyaLagCntTrend{})
}

// 虎牙卡顿样本数趋势图

type HuyaLagCntTrend struct {
}

func (h *HuyaLagCntTrend) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" && req.AppName != "huyap2p" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.LagCnt == nil {
				continue
			}
			cb(*report.Ts_m, *report.LagCnt)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙卡顿样本数数趋势图", "卡顿样本", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaLagCntTrend) ID() string {
	return "chart_hyLagCntTrend"
}

func (h *HuyaLagCntTrend) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙卡顿样本数数趋势图",
	}
}

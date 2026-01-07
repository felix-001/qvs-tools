package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&P2pPercent{})
}

// p2p样本占总样本的比重趋势图

type P2pPercent struct {
}

func (h *P2pPercent) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" && req.AppName != "huyap2p" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	req.Protocol = ""
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.P2pPercent == nil {
				continue
			}
			cb(*report.Ts_m, *report.P2pPercent)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙p2p样本占总样本数的比重", "占比", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *P2pPercent) ID() string {
	return "chart_p2p_percent"
}

func (h *P2pPercent) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙p2p样本占总样本数的比重",
	}
}

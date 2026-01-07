package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&Huyap2pLagRate{})
}

// 虎牙p2p每分钟卡顿率趋势图

type Huyap2pLagRate struct {
}

func (h *Huyap2pLagRate) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" && req.AppName != "huyap2p" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	req.Protocol = "p2p"
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.Percent == nil {
				continue
			}
			cb(*report.Ts_m, *report.Percent)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙p2p整体每分钟卡顿率趋势图", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *Huyap2pLagRate) ID() string {
	return "chart_hyp2pLagRateTrend"
}

func (h *Huyap2pLagRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙p2p整体卡顿率趋势图",
	}
}

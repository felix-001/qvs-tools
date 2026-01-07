package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&HuyaflvLagRate{})
}

// 虎牙flv每分钟卡顿率趋势图

type HuyaflvLagRate struct {
}

func (h *HuyaflvLagRate) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" && req.AppName != "huyap2p" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	req.Protocol = "flv"
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
	chartData := qos.GenerateLineChartData("虎牙flv整体每分钟卡顿率趋势图", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HuyaflvLagRate) ID() string {
	return "chart_hyflvLagRateTrend"
}

func (h *HuyaflvLagRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙flv整体卡顿率趋势图",
	}
}

package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&PatchLagPercent{})
}

// 虎牙补片卡顿样本占总p2p卡顿样本的比例

type PatchLagPercent struct {
}

func (h *PatchLagPercent) Generate(req qos.QOSRequest) any {
	req.Protocol = "p2p"
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.PatchLagPercent == nil {
				continue
			}
			cb(*report.Ts_m, *report.PatchLagPercent)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙补片卡顿样本占总p2p卡顿样本的比例", "比重", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *PatchLagPercent) ID() string {
	return "chart_hyPatchLagPercent"
}

func (h *PatchLagPercent) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙补片卡顿样本占总p2p卡顿样本的比例",
	}
}

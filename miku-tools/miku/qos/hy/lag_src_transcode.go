package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&LagSrcTranscodeRate{})
}

// 虎牙回客户源站转码流卡顿率

type LagSrcTranscodeRate struct {
}

func (l *LagSrcTranscodeRate) Generate(req qos.QOSRequest) any {
	hyLagRateReports, err := qos.GetHyLagRateReports(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyLagRateReports:", len(hyLagRateReports))
	fn := func(cb qos.Cb) {
		for _, report := range hyLagRateReports {
			if report.SrcTranscodeLagRate == nil {
				continue
			}
			cb(*report.Ts_m, *report.SrcTranscodeLagRate)
		}
	}
	chartData := qos.GenerateLineChartData("虎牙回客户源站转码流卡顿率", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (l *LagSrcTranscodeRate) ID() string {
	return "chart_lagSrcTranscodePercent"
}

func (l *LagSrcTranscodeRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    l.ID(),
		Title: "虎牙回客户源站转码流卡顿率",
	}
}

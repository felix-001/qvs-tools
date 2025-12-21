package qos

import (
	"log"
)

// 秒开率, SendFirstPktTime

func init() {
	RegisterChartGenerator(&LoadRate{})
}

type LoadRate struct {
}

func (l *LoadRate) ID() string {
	return "chart_loadRate"
}

func (l *LoadRate) ChartInfo() ChartInfo {
	return ChartInfo{
		ID:    l.ID(),
		Title: "秒开率",
	}
}

func (l *LoadRate) Generate(req QOSRequest) any {
	reports, err := GetMikuStreamdReportLagDatas(req)
	log.Println("查询结果reports:", len(reports))
	if err != nil {
		return nil
	}
	fn := func(cb Cb) {
		for _, report := range reports {
			if report.LoadRatio == nil {
				continue
			}
			cb(*report.Ts_m, *report.LoadRatio)
		}
	}
	chartData := GenerateLineChartData("秒开率", "秒开率", fn)
	return ChartData{Data: chartData, Type: ChartTypeLine}
}

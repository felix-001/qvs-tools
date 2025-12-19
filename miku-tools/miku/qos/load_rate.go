package qos

import (
	"log"
)

// 秒开率, SendFirstPktTime

func init() {
	RegisterChartGenerator("loadRate", &LoadRate{})
}

type LoadRate struct {
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
	return chartData
}

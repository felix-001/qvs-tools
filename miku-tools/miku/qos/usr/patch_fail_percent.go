package usr

import (
	"log"
	"mikutool/miku/qos"
)

// 补片失败率

func init() {
	qos.RegisterChartGenerator(&PatchFailPercent{})
}

type PatchFailPercent struct {
}

func (c *PatchFailPercent) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyap2p" {
		log.Println("appName is not huyap2p")
		return qos.ChartData{Data: "", Type: "error"}
	}
	log.Println("Generating patch fail percent chart")
	streamdReports, err := qos.GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return qos.ChartData{Data: "", Type: "error"}
	}
	log.Println("streamdReports:", len(streamdReports))
	fn := func(cb qos.Cb) {
		for _, report := range streamdReports {
			if report.Patch_fail_percent == nil {
				continue
			}
			cb(*report.Ts_m, *report.Patch_fail_percent)
		}
	}
	chartData := qos.GenerateLineChartData("补片失败率", "失败率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (c *PatchFailPercent) ID() string {
	return "chart_patchFailPercent"
}

func (c *PatchFailPercent) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    c.ID(),
		Title: "补片失败率",
	}
}

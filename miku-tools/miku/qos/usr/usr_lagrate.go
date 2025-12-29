package usr

import (
	"log"
	"mikutool/miku/qos"
)

// 用户百秒卡顿率

func init() {
	qos.RegisterChartGenerator(&UsrLagRate{})
}

type UsrLagRate struct {
}

func (u *UsrLagRate) ID() string {
	return "chart_usrLagRate"
}

func (u *UsrLagRate) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    u.ID(),
		Title: "用户整体百秒卡顿率",
	}
}

func (u *UsrLagRate) Generate(req qos.QOSRequest) any {
	streamdReports, err := qos.GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return qos.ChartData{Data: "", Type: "error"}
	}
	fn := func(cb qos.Cb) {
		for _, report := range streamdReports {
			if report.Ratio_lag_player == nil {
				continue
			}
			cb(*report.Ts_m, *report.Ratio_lag_player)
		}
	}
	chartData := qos.GenerateLineChartData("用户整体百秒卡顿率", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

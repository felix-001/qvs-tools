package usr

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

func init() {
	qos.RegisterChartGenerator(&MikuOnlineNum{})
}

type MikuOnlineNum struct {
}

func (s *MikuOnlineNum) ID() string {
	return "chart_onlineNum"
}

func (s *MikuOnlineNum) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    s.ID(),
		Title: "在线用户数趋势图",
	}
}

func (s *MikuOnlineNum) Generate(req qos.QOSRequest) any {
	reports, err := qos.GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Printf("获取数据失败: %v", err)
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("reports:", len(reports))
	fn := func(cb qos.Cb) {
		for _, report := range reports {
			if report.UsrCnt == nil {
				continue
			}
			cb(*report.Ts_m, *report.UsrCnt)
		}
	}
	chartData := qos.GenerateLineChartData("在线用户数趋势图", "用户数", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

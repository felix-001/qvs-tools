package upstream

import (
	"log"
	"mikutool/miku/qos"
)

// 回客户源站重试次数

func init() {
	qos.RegisterChartGenerator(&CustomerUpstreamRetryTimes{})
}

type CustomerUpstreamRetryTimes struct {
}

func (c *CustomerUpstreamRetryTimes) Generate(req qos.QOSRequest) any {
	log.Println("Generating customer upstream retry times chart")
	streamdReports, err := qos.GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	log.Println("streamdReports:", len(streamdReports))
	fn := func(cb qos.Cb) {
		for _, report := range streamdReports {
			if report.UpstreamRetryTimes == nil {
				continue
			}
			cb(*report.Ts_m, *report.UpstreamRetryTimes)
		}
	}
	chartData := qos.GenerateLineChartData("回客户源站重试次数", "重试次数", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (c *CustomerUpstreamRetryTimes) ID() string {
	return "chart_customerUpstreamRetryTimes"
}

func (c *CustomerUpstreamRetryTimes) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    c.ID(),
		Title: "回客户源站重试次数",
	}
}

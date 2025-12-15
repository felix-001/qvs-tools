package qos

import (
	"log"
)

func init() {
	RegisterChartGenerator("miku_lag_rate", &MikuLagRate{})
}

type MikuLagRate struct {
}

func (m *MikuLagRate) Generate(req QOSRequest) any {
	return ""
	streamdReports, err := GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	log.Println("查询结果streamdReports:", len(streamdReports))
	streamdChartsHTML := GenerateStreamdChartsHTML(streamdReports)
	return streamdChartsHTML
}

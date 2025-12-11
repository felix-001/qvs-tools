package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"
)

func init() {
	RegisterChartGenerator("miku_lag_rate", &MikuLagRate{})
}

type MikuLagRate struct {
}

func (m *MikuLagRate) Generate(req QOSRequest) string {
	var streamdReports []util.StreamdLagReport
	sql := BuildMikuSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &streamdReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果streamdReports:", len(streamdReports))
	streamdChartsHTML := GenerateStreamdChartsHTML(streamdReports)
	return streamdChartsHTML
}

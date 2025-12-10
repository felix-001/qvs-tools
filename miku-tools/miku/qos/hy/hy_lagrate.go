package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

func init() {
	log.Println("init hy_lagrate")
	qos.RegisterChartGenerator("hy_lagrate", &HyLagRate{})
	qos.RegisterChartGenerator("miku_lagrate", &MikuLagRate{})
}

type HyLagRate struct {
}

func (h *HyLagRate) Generate(req qos.QOSRequest) string {
	// 构建SQL查询（获取原始数据）
	sql := qos.BuildSQLQuery(req, true)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}

	// 执行查询
	var reports []util.QualityReport
	if err := util.TrinoQuery("miku", sql, &reports); err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果:", len(reports))
	if req.RawData {
		qos.SaveCsv(reports)
	}

	_, clientAggregated, cdnAggData, _ := aggregateReport(reports, req.IpParser)

	// 生成聚合数据表格HTML
	tableHTML := qos.GenerateTableHTML([]qos.AggregatedData{}, clientAggregated)

	// 生成CDN卡顿用户表格HTML
	cdnLagTableHTML := qos.GenerateCDNLagTableHTML(cdnAggData)

	return tableHTML + cdnLagTableHTML
}

type MikuLagRate struct {
}

func (m *MikuLagRate) Generate(req qos.QOSRequest) string {
	var streamdReports []util.StreamdLagReport
	sql := qos.BuildMikuSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &streamdReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果streamdReports:", len(streamdReports))
	streamdChartsHTML := qos.GenerateStreamdChartsHTML(streamdReports)
	return streamdChartsHTML
}

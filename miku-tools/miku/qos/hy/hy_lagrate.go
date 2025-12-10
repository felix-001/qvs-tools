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

	// 在Go代码中实现按分钟聚合（模拟第二个SQL查询的效果）
	minuteAggregated := aggregateByMinute(reports)

	//cdnAggregated, clientAggregated, cdnAggData, aggData := aggregateReport(reports, req.IpParser)
	_, clientAggregated, cdnAggData, aggData := aggregateReport(reports, req.IpParser)

	// 聚合在线用户数（使用示例日期和小时，实际应该从请求参数获取）
	onlineUsersAggregated := aggregateOnlineUsers(reports)

	// 生成分钟聚合数据的折线图
	//minuteChartHTML := generateMinuteChartHTML(minuteAggregated)

	// 生成卡顿用户占比折线图
	//lagUserRatioChartHTML := qos.GenerateLagUserRatioChartHTML(minuteAggregated)

	// 生成节点卡顿占比折线图
	cdnLagRatioChartHTML := qos.GenerateCdnLagRatioChartHTML(minuteAggregated)

	// 生成在线用户数折线图
	onlineUsersChartHTML := qos.GenerateOnlineUsersChartHTML(onlineUsersAggregated)

	// 生成聚合数据表格HTML
	//tableHTML := generateTableHTML(cdnAggregated, clientAggregated)
	tableHTML := qos.GenerateTableHTML([]qos.AggregatedData{}, clientAggregated)

	// 生成CDN卡顿用户表格HTML
	cdnLagTableHTML := qos.GenerateCDNLagTableHTML(cdnAggData)

	// 生成聚合数据饼图HTML
	aggDataChartsHTML := qos.GenerateAggDataChartsHTML(aggData)

	//return minuteChartHTML + lagUserRatioChartHTML + cdnLagRatioChartHTML + onlineUsersChartHTML + tableHTML + cdnLagTableHTML + aggDataChartsHTML
	return cdnLagRatioChartHTML + onlineUsersChartHTML + tableHTML + cdnLagTableHTML + aggDataChartsHTML
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

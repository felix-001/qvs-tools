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

	_, _, cdnAggData, _ := aggregateReport(reports, req.IpParser)

	// 生成CDN卡顿用户表格HTML
	cdnLagTableHTML := qos.GenerateCDNLagTableHTML(cdnAggData)

	return cdnLagTableHTML
}

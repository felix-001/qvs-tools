package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"
)

func init() {
	RegisterChartGenerator("hy_cdn_lag", &HyCdnLag{})
}

type HyCdnLag struct {
}

func (h *HyCdnLag) Generate(req QOSRequest) string {
	sql := buildHyCdnLagSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyCdnLagReports []util.HyCdnLagReport
	if err := util.TrinoQuery("miku", sql, &hyCdnLagReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果hyCdnLagReports:", len(hyCdnLagReports))
	if req.RawData {
		saveHyCdnLagRawData(hyCdnLagReports)
	}
	return ""
}

package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"
)

// 推流/回源带宽折线图

func init() {
	RegisterChartGenerator("upstreamBandwidth", &UpstreamBandwidth{})
}

type UpstreamBandwidth struct {
}

func (u *UpstreamBandwidth) Generate(req QOSRequest) any {
	if req.StreamID == "" || req.FuzzySearch {
		return ""
	}
	var upstreamBandwidthReports []util.StreamdUpstreamBandWidthReport
	sql := buildMikuUpstreamBandwidthSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &upstreamBandwidthReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果upstreamBandwidthReports:", len(upstreamBandwidthReports))
	fn := func(cb Cb) {
		for _, report := range upstreamBandwidthReports {
			if report.BandWidth == nil {
				continue
			}
			cb(*report.Ts, *report.BandWidth)
		}
	}
	chartData := GenerateLineChartData("推流/回源带宽趋势图", "带宽", fn)
	return chartData
	// 生成推流/回源带宽折线图
	upstreamBandwidthChartHTML := generateUpstreamBandwidthChartHTML(upstreamBandwidthReports)
	return upstreamBandwidthChartHTML
}

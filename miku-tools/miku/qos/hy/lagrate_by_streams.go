package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

// 卡顿率按流分布

func init() {
	qos.RegisterChartGenerator(&LagRateByStreams{})
}

type LagRateByStreams struct {
}

func (l *LagRateByStreams) getStreamClientIpsMap(clientCdnIps []util.HyClientIpsOnCdnIpReport) (map[string][]string, map[string][]string) {
	lagClientIpsMap := make(map[string][]string)
	normalClientIpsMap := make(map[string][]string)
	for _, report := range clientCdnIps {
		if report.DimIp == nil || report.LagCnt == nil || report.StreamName == nil {
			continue
		}
		if *report.LagCnt > 0 {
			if _, ok := lagClientIpsMap[*report.StreamName]; !ok {
				lagClientIpsMap[*report.StreamName] = make([]string, 0)
			}
			lagClientIpsMap[*report.StreamName] = append(lagClientIpsMap[*report.StreamName], *report.DimIp)
		} else {
			if _, ok := normalClientIpsMap[*report.StreamName]; !ok {
				normalClientIpsMap[*report.StreamName] = make([]string, 0)
			}
			normalClientIpsMap[*report.StreamName] = append(normalClientIpsMap[*report.StreamName], *report.DimIp)
		}
	}
	return lagClientIpsMap, normalClientIpsMap
}

func (l *LagRateByStreams) Generate(req qos.QOSRequest) any {
	if req.AppName != "huyacdn" && req.AppName != "huyap2p" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	sql := qos.BuildHyLagRateByStreamsSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyLagRateByStreamsReports []util.HyLagRateByStreamsReport
	if err := util.TrinoQuery("miku", sql, &hyLagRateByStreamsReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return ""
	}
	clientCdnIps, err := qos.GetHyDistinctCdnClientIps(req)
	if err != nil {
		log.Printf("获取CDN客户端IP失败: %v", err)
		return qos.ChartData{Data: fmt.Sprintf("获取CDN客户端IP失败: %v", err), Type: "error"}
	}

	lagClientIpsMap, normalClientIpsMap := l.getStreamClientIpsMap(clientCdnIps)

	log.Println("len hyLagRateByStreamsReports", len(hyLagRateByStreamsReports))
	var lagTotal int
	for _, report := range hyLagRateByStreamsReports {
		lagTotal += *report.LagCnt
	}
	for i, report := range hyLagRateByStreamsReports {
		if report.LagCnt == nil || report.StreamName == nil {
			continue
		}
		weight := float64(*report.LagCnt) * 100.0 / float64(lagTotal)
		hyLagRateByStreamsReports[i].Weight = &weight
		if _, ok := lagClientIpsMap[*report.StreamName]; ok {
			hyLagRateByStreamsReports[i].LagClientIps = lagClientIpsMap[*report.StreamName]
		}
		if _, ok := normalClientIpsMap[*report.StreamName]; ok {
			hyLagRateByStreamsReports[i].NormalClientIps = normalClientIpsMap[*report.StreamName]
		}
		if len(lagClientIpsMap[*report.StreamName])+len(normalClientIpsMap[*report.StreamName]) > 0 {
			hyLagRateByStreamsReports[i].LagClientRate = float64(len(lagClientIpsMap[*report.StreamName])) * 100.0 / float64(len(lagClientIpsMap[*report.StreamName])+len(normalClientIpsMap[*report.StreamName]))
		}
	}
	tableData := map[string]any{
		"title": "虎牙按流卡顿率",
		"data":  hyLagRateByStreamsReports,
	}
	return qos.ChartData{Data: tableData, Type: qos.ChartTypeTable}
}

func (l *LagRateByStreams) ID() string {
	return "chart_hyLagrateByStreams"
}

func (l *LagRateByStreams) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    l.ID(),
		Title: "虎牙按流卡顿率表格",
	}
}

package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

func init() {
	qos.RegisterChartGenerator(&LagRateByUser{})
}

// 每个用户的卡顿率表格
type LagRateByUser struct {
}

func (l *LagRateByUser) getclientCdnIpsMap(clientCdnIps []util.HyClientIpsOnCdnIpReport) map[string]map[string]bool {
	clientCdnsMap := make(map[string]map[string]bool)
	for _, report := range clientCdnIps {
		if report.DimCdnip == nil || report.DimIp == nil {
			continue
		}
		if _, ok := clientCdnsMap[*report.DimIp]; !ok {
			clientCdnsMap[*report.DimIp] = make(map[string]bool)
		}
		clientCdnsMap[*report.DimIp][*report.DimCdnip] = true
	}
	return clientCdnsMap
}

func (l *LagRateByUser) getStreams(clientCdnIps []util.HyClientIpsOnCdnIpReport) (map[string][]string, map[string][]string) {
	normalStreamMap := make(map[string]map[string]bool)
	lagStreamMap := make(map[string]map[string]bool)

	// 去重
	for _, report := range clientCdnIps {
		if report.DimIp == nil || report.StreamName == nil || report.LagCnt == nil {
			continue
		}
		if *report.LagCnt > 0 {
			if _, ok := lagStreamMap[*report.DimIp]; !ok {
				lagStreamMap[*report.DimIp] = make(map[string]bool)
			}
			lagStreamMap[*report.DimIp][*report.StreamName] = true
		} else {
			if _, ok := normalStreamMap[*report.DimIp]; !ok {
				normalStreamMap[*report.DimIp] = make(map[string]bool)
			}
			normalStreamMap[*report.DimIp][*report.StreamName] = true
		}
	}

	normalStreamsMap := make(map[string][]string)
	lagStreamsMap := make(map[string][]string)
	for clientIp, streamMap := range normalStreamMap {
		normalStreamsMap[clientIp] = make([]string, 0)
		for stream := range streamMap {
			normalStreamsMap[clientIp] = append(normalStreamsMap[clientIp], stream)
		}
	}
	for clientIp, streamMap := range lagStreamMap {
		lagStreamsMap[clientIp] = make([]string, 0)
		for stream := range streamMap {
			lagStreamsMap[clientIp] = append(lagStreamsMap[clientIp], stream)
		}
	}
	return normalStreamsMap, lagStreamsMap
}
func (l *LagRateByUser) splitCdnIps(clientCdnsMap map[string]map[string]bool, report util.HyCdnLagReport) ([]string, []string) {
	var normalCdnIps []string
	var lagCdnIps []string
	if cdnMap, ok := clientCdnsMap[*report.DimIp]; ok {
		for cdnIp := range cdnMap {
			if *report.LagCnt > 0 {
				lagCdnIps = append(lagCdnIps, cdnIp)
			} else {
				normalCdnIps = append(normalCdnIps, cdnIp)
			}
		}
	}
	return normalCdnIps, lagCdnIps
}

func (l *LagRateByUser) getAggData(reports []util.HyCdnLagReport, req qos.QOSRequest, clientCdnIps []util.HyClientIpsOnCdnIpReport) qos.ChartData {

	aggData := make([]qos.UserAggregatedData, 0)
	totalLagCnt := 0
	clientCdnIpsMap := l.getclientCdnIpsMap(clientCdnIps)
	normalStreamsMap, lagStreamsMap := l.getStreams(clientCdnIps)
	for _, report := range reports {
		if report.DimIp == nil || report.LagCnt == nil || report.Total == nil {
			continue
		}
		normalCdnIps, lagCdnIps := l.splitCdnIps(clientCdnIpsMap, report)
		_, isp, _, prov := util.GetLocate(*report.DimIp, req.IpParser)
		totalNodeCnt := len(normalCdnIps) + len(lagCdnIps)
		lagNodeRate := 0.0
		if totalNodeCnt > 0 {
			lagNodeRate = float64(len(lagCdnIps)) * 100.0 / float64(totalNodeCnt)
		}
		aggData = append(aggData, qos.UserAggregatedData{
			ClientIP:      *report.DimIp,
			LagCount:      *report.LagCnt,
			TotalCount:    *report.Total,
			Percent:       float64(*report.LagCnt) * 100 / float64(*report.Total),
			Isp:           isp,
			Prov:          prov,
			NormalIps:     normalCdnIps,
			LagNodeIps:    lagCdnIps,
			LagNodeCnt:    len(lagCdnIps),
			TotalNodeCnt:  totalNodeCnt,
			LagNodeRate:   lagNodeRate,
			NormalStreams: normalStreamsMap[*report.DimIp],
			LagStreams:    lagStreamsMap[*report.DimIp],
		})
		totalLagCnt += *report.LagCnt
	}
	var totalWight float64
	cnt := 0
	total := 0
	for i, data := range aggData {
		data.Weight = float64(data.LagCount*100) / float64(totalLagCnt)
		aggData[i] = data
		totalWight += data.Weight
		if data.Weight > 0 {
			cnt++
		}
		total++
	}
	log.Println("totalWight:", totalWight, "cnt:", cnt, "total:", total)
	return qos.ChartData{Data: aggData, Type: qos.ChartTypeTable}
}

func (l *LagRateByUser) Generate(req qos.QOSRequest) any {
	sql := qos.BuildHyClientLagSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Println("lag_rate_by_user sql:", sql)
	}

	var reports []util.HyCdnLagReport
	if err := util.TrinoQuery("miku", sql, &reports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("lag_rate_by_user reports:", len(reports))
	clientCdnIps, err := qos.GetHyDistinctCdnClientIps(req)
	if err != nil {
		log.Printf("获取CDN客户端IP失败: %v", err)
		return qos.ChartData{Data: fmt.Sprintf("获取CDN客户端IP失败: %v", err), Type: "error"}
	}
	aggData := l.getAggData(reports, req, clientCdnIps)
	return aggData
}

func (l *LagRateByUser) ID() string {
	return "chart_lagrateByUser"
}

func (l *LagRateByUser) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    l.ID(),
		Title: "虎牙按用户卡顿分析表格",
	}
}

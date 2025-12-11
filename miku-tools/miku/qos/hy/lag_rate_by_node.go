package hy

import (
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"

	"github.com/qbox/pili/common/ipdb.v1"
)

// 每个节点的卡顿率

func init() {
	log.Println("init hy_cdn_lag")
	qos.RegisterChartGenerator("hy_cdn_lag", &HyCdnLag{})
}

type HyCdnLag struct {
	ipparser *ipdb.City
}

func (h *HyCdnLag) getRawData(req qos.QOSRequest) []util.HyCdnLagReport {
	sql := qos.BuildHyCdnLagSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyCdnLagReports []util.HyCdnLagReport
	if err := util.TrinoQuery("miku", sql, &hyCdnLagReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return nil
	}
	log.Println("查询结果hyCdnLagReports:", len(hyCdnLagReports))
	if req.RawData {
		qos.SaveHyCdnLagRawData(hyCdnLagReports)
	}
	return hyCdnLagReports
}

func (h *HyCdnLag) getClientIpsOnCdnIp(req qos.QOSRequest) ([]util.HyClientIpsOnCdnIpReport, error) {
	hyClientIpsOnCdnIpReports, err := qos.GetHyDistinctCdnClientIps(req)
	log.Println("查询结果hyClientIpsOnCdnIpReports:", len(hyClientIpsOnCdnIpReports))
	if req.RawData {
		qos.SaveHyClientIpsOnCdnIpRawData(hyClientIpsOnCdnIpReports)
	}
	return hyClientIpsOnCdnIpReports, err
}

func (h *HyCdnLag) aggCdnLagData(hyCdnLagReports []util.HyCdnLagReport, clientIpsOnCdnIpReports []util.HyClientIpsOnCdnIpReport) []qos.AggregatedData {
	totalLagCnt := 0
	for _, report := range hyCdnLagReports {
		if report.LagCnt == nil || report.DimCdnip == nil || report.Total == nil {
			continue
		}
		totalLagCnt += *report.LagCnt
	}

	// cdn ip ->  client ips
	var cdnIp2ClientipsMap = make(map[string]map[string]bool)
	for _, report := range clientIpsOnCdnIpReports {
		if report.DimCdnip == nil || report.DimIp == nil {
			continue
		}
		if _, ok := cdnIp2ClientipsMap[*report.DimCdnip]; !ok {
			cdnIp2ClientipsMap[*report.DimCdnip] = make(map[string]bool)
		}
		cdnIp2ClientipsMap[*report.DimCdnip][*report.DimIp] = true
	}

	var aggCdnLagDatas []qos.AggregatedData
	for _, report := range hyCdnLagReports {
		if report.LagCnt == nil || report.DimCdnip == nil || report.Total == nil {
			continue
		}
		cdnIp := *report.DimCdnip
		var clientIps []string
		if _, ok := cdnIp2ClientipsMap[cdnIp]; ok {
			for clientIp := range cdnIp2ClientipsMap[cdnIp] {
				clientIps = append(clientIps, clientIp)
			}
		}
		_, isp, _, prov := util.GetLocate(cdnIp, h.ipparser)
		aggCdnLagDatas = append(aggCdnLagDatas, qos.AggregatedData{
			IP:         cdnIp,
			LagCount:   *report.LagCnt,
			TotalCount: *report.Total,
			Isp:        isp,
			Prov:       prov,
			RemoteIps:  clientIps,
			LagRate:    float64(*report.LagCnt*100) / float64(totalLagCnt),
		})
	}
	return aggCdnLagDatas
}

func (h *HyCdnLag) Generate(req qos.QOSRequest) string {
	h.ipparser = req.IpParser
	hyCdnLagReports := h.getRawData(req)
	clientIpsOnCdnIpReports, err := h.getClientIpsOnCdnIp(req)
	if err != nil {
		log.Printf("获取客户端IP聚合数据失败: %v", err)
		return ""
	}
	aggCdnLagData := h.aggCdnLagData(hyCdnLagReports, clientIpsOnCdnIpReports)
	//log.Println("aggCdnLagData:", aggCdnLagData)
	tableHTML := qos.GenerateTableHTML(aggCdnLagData, []qos.AggregatedData{})
	return tableHTML
}

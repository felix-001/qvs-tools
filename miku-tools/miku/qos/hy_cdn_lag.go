package qos

import (
	"log"
	"mikutool/public/util"

	"github.com/qbox/pili/common/ipdb.v1"
)

func init() {
	RegisterChartGenerator("hy_cdn_lag", &HyCdnLag{})
}

type HyCdnLag struct {
	ipparser *ipdb.City
}

func (h *HyCdnLag) getRawData(req QOSRequest) []util.HyCdnLagReport {
	sql := buildHyCdnLagSQLQuery(req)
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
		saveHyCdnLagRawData(hyCdnLagReports)
	}
	return hyCdnLagReports
}

func (h *HyCdnLag) getClientIpsOnCdnIp(req QOSRequest) []util.HyClientIpsOnCdnIpReport {
	sql := buildClientIpsOnCdnIpsSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyClientIpsOnCdnIpReports []util.HyClientIpsOnCdnIpReport
	if err := util.TrinoQuery("miku", sql, &hyClientIpsOnCdnIpReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return nil
	}
	log.Println("查询结果hyClientIpsOnCdnIpReports:", len(hyClientIpsOnCdnIpReports))
	if req.RawData {
		saveHyClientIpsOnCdnIpRawData(hyClientIpsOnCdnIpReports)
	}
	return hyClientIpsOnCdnIpReports
}

func (h *HyCdnLag) aggCdnLagData(hyCdnLagReports []util.HyCdnLagReport, clientIpsOnCdnIpReports []util.HyClientIpsOnCdnIpReport) []AggregatedData {
	totalLagCnt := 0
	for _, report := range hyCdnLagReports {
		if report.LagCnt == nil {
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

	var aggCdnLagDatas []AggregatedData
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
		aggCdnLagDatas = append(aggCdnLagDatas, AggregatedData{
			IP:         cdnIp,
			LagCount:   *report.LagCnt,
			TotalCount: *report.Total,
			Isp:        isp,
			Prov:       prov,
			RemoteIps:  clientIps,
			LagRate:    float64(*report.LagCnt*100) / float64(*report.Total),
		})
	}
	return aggCdnLagDatas
}

func (h *HyCdnLag) Generate(req QOSRequest) string {
	h.ipparser = req.IpParser
	hyCdnLagReports := h.getRawData(req)
	clientIpsOnCdnIpReports := h.getClientIpsOnCdnIp(req)
	aggCdnLagData := h.aggCdnLagData(hyCdnLagReports, clientIpsOnCdnIpReports)
	//log.Println("aggCdnLagData:", aggCdnLagData)
	tableHTML := generateTableHTML(aggCdnLagData, []AggregatedData{})
	return tableHTML
}

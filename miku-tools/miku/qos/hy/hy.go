package hy

import (
	"mikutool/miku/qos"
	"mikutool/public/util"
	"net"
	"net/url"
	"sort"

	"github.com/qbox/pili/common/ipdb.v1"
)

func aggregateReport(reports []util.QualityReport, ipparser *ipdb.City) ([]qos.AggregatedData, []qos.AggregatedData, []qos.CdnAggregateData, qos.AggData) {
	// CDN IP聚合
	cdnLagCntMap := make(map[string]int)   // cdnip -> 延迟数
	cdnTotalCntMap := make(map[string]int) // cdnip -> 总数
	cdnLagClientMap := make(map[string]map[string]bool)
	cdnAllClientMap := make(map[string]map[string]bool)

	// Client IP聚合
	clientLagCntMap := make(map[string]int)   // clientIp -> 延迟数
	clientTotalCntMap := make(map[string]int) // clientIp -> 总数
	clientCdnIpsMap := make(map[string]map[string]bool)

	areaLagCntMap := make(map[string]int) // area -> 延迟数
	provLagCntMap := make(map[string]int) // prov -> 延迟数

	for _, report := range reports {
		// 获取CDN IP
		var cdnip string
		if report.DimCdnip != nil && *report.DimCdnip != "" {
			cdnip = *report.DimCdnip
		}

		if (cdnip == "" || cdnip == "qn.flv.huya.com" || cdnip == "http://qn.flv.huya.com") &&
			report.DimStreamUrl != nil && *report.DimStreamUrl != "" {
			// 从URL中提取域名作为CDN IP
			if u, err := url.Parse(*report.DimStreamUrl); err == nil {
				host := u.Host
				if h, _, err := net.SplitHostPort(host); err == nil {
					cdnip = h
				} else {
					cdnip = host
				}
			}
		}

		// 获取Client IP
		var clientIp string
		if report.DimIp != nil {
			clientIp = *report.DimIp
		}

		// 获取不良质量状态
		var isBadQuality bool
		if report.FieldVideoBadQuality != nil && *report.FieldVideoBadQuality == 100 {
			isBadQuality = true
		}

		// 处理CDN IP聚合
		if cdnip != "" {
			cdnTotalCntMap[cdnip]++
			if isBadQuality {
				cdnLagCntMap[cdnip]++
			}
		}

		// 处理Client IP聚合
		if clientIp != "" {
			clientTotalCntMap[clientIp]++
			if isBadQuality {
				clientLagCntMap[clientIp]++
				_, _, area, prov := util.GetLocate(clientIp, ipparser)
				areaLagCntMap[area]++
				provLagCntMap[prov]++

			}
			if _, exists := clientCdnIpsMap[clientIp]; !exists {
				clientCdnIpsMap[clientIp] = make(map[string]bool)
			}
			clientCdnIpsMap[clientIp][cdnip] = true
		}

		if isBadQuality {
			if _, exists := cdnLagClientMap[cdnip]; !exists {
				cdnLagClientMap[cdnip] = make(map[string]bool)
			}
			cdnLagClientMap[cdnip][clientIp] = true
		}
		if _, exists := cdnAllClientMap[cdnip]; !exists {
			cdnAllClientMap[cdnip] = make(map[string]bool)
		}
		cdnAllClientMap[cdnip][clientIp] = true
	}

	// 生成CDN聚合数据
	var cdnAggregated []qos.AggregatedData
	totalLagCdnCnt := 0
	for cdnip, total := range cdnTotalCntMap {
		if cdnip == "qn.flv.huya.com" {
			continue
		}
		clientIps := make([]string, 0)
		for clientIp := range cdnAllClientMap[cdnip] {
			clientIps = append(clientIps, clientIp)
		}
		_, isp, _, prov := util.GetLocate(cdnip, ipparser)
		cdnAggregated = append(cdnAggregated, qos.AggregatedData{
			IP:         cdnip,
			TotalCount: total,
			LagCount:   cdnLagCntMap[cdnip],
			Prov:       prov,
			Isp:        isp,
			NormalIps:  clientIps,
		})
		totalLagCdnCnt += cdnLagCntMap[cdnip]
	}
	for i := range cdnAggregated {
		cdnAggregated[i].LagRate = float64(cdnAggregated[i].LagCount*100) / float64(totalLagCdnCnt)
	}

	// 按延迟次数降序排序
	sort.Slice(cdnAggregated, func(i, j int) bool {
		return cdnAggregated[i].LagCount > cdnAggregated[j].LagCount
	})

	// 生成Client聚合数据
	var clientAggregated []qos.AggregatedData
	var aggData qos.AggData
	clientCountryCntMap := make(map[string]int)
	areaCntMap := make(map[string]int)
	provCntMap := make(map[string]int)
	totalLagCnt := 0
	for clientIp, total := range clientTotalCntMap {
		cdnIps := make([]string, 0)
		for cdnip := range clientCdnIpsMap[clientIp] {
			cdnIps = append(cdnIps, cdnip)
		}
		country, isp, area, prov := util.GetLocate(clientIp, ipparser)
		clientAggregated = append(clientAggregated, qos.AggregatedData{
			IP:         clientIp,
			TotalCount: total,
			LagCount:   clientLagCntMap[clientIp],
			Prov:       prov,
			Isp:        isp,
			NormalIps:  cdnIps,
		})
		totalLagCnt += clientLagCntMap[clientIp]
		clientCountryCntMap[country]++
		areaCntMap[area]++
		provCntMap[prov]++
	}
	for i := range clientAggregated {
		clientAggregated[i].LagRate = float64(clientAggregated[i].LagCount*100) / float64(totalLagCnt)
	}
	aggData.CountryCntMap = clientCountryCntMap
	aggData.AreaCntMap = areaCntMap
	aggData.ProvCntMap = provCntMap
	aggData.AreaLagCntMap = areaLagCntMap
	aggData.ProvLagCntMap = provLagCntMap

	// 按不良质量次数降序排序
	sort.Slice(clientAggregated, func(i, j int) bool {
		return clientAggregated[i].LagCount > clientAggregated[j].LagCount
	})

	var cdnAggregateData []qos.CdnAggregateData
	for cdnip, clientsMap := range cdnLagClientMap {
		var lagIps []string
		for clientIp := range clientsMap {
			lagIps = append(lagIps, clientIp)
		}
		cdnAggregateData = append(cdnAggregateData, qos.CdnAggregateData{
			IP:             cdnip,
			TotalUserCount: len(cdnAllClientMap[cdnip]),
			LagIps:         lagIps,
		})
	}

	return cdnAggregated, clientAggregated, cdnAggregateData, aggData
}

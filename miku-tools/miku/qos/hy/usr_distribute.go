package hy

import (
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

// 用户按国家、大区、省份分布

func init() {
	qos.RegisterChartGenerator("usr_distribute", &UsrDistribute{})
}

type UsrDistribute struct {
}

func (u *UsrDistribute) getAggData(req qos.QOSRequest, reports []util.HyClientIpsOnCdnIpReport) qos.AggData {
	clientIpMap := make(map[string]bool)
	for _, report := range reports {
		if report.DimIp == nil {
			continue
		}
		clientIpMap[*report.DimIp] = true
	}
	countryMap := make(map[string]int)
	areaMap := make(map[string]int)
	provinceMap := make(map[string]int)
	for clientIp := range clientIpMap {
		country, _, area, province := util.GetLocate(clientIp, req.IpParser)
		countryMap[country]++
		areaMap[area]++
		provinceMap[province]++
	}
	aggData := qos.AggData{
		CountryCntMap: countryMap,
		AreaCntMap:    areaMap,
		ProvCntMap:    provinceMap,
	}
	return aggData
}

func (u *UsrDistribute) Generate(req qos.QOSRequest) string {
	cdnClientIps, err := qos.GetHyDistinctCdnClientIps(req)
	if err != nil {
		log.Printf("获取CDN客户端IP失败: %v", err)
		return ""
	}
	log.Println("cdnClientIps:", len(cdnClientIps))
	aggData := u.getAggData(req, cdnClientIps)
	return qos.GenerateAggDataChartsHTML(aggData)
}

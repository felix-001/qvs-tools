package hy

import (
	"log"
	"mikutool/miku/qos"
)

// 用户按国家、大区、省份分布

func init() {
	qos.RegisterChartGenerator(&UsrDistribute{})
}

type UsrDistribute struct {
}

func (u *UsrDistribute) Generate(req qos.QOSRequest) any {
	log.Printf("req: %+v", req)
	cdnClientIps, err := qos.GetHyDistinctCdnClientIps(req)
	if err != nil {
		log.Printf("获取CDN客户端IP失败: %v", err)
		return qos.ChartData{Data: "", Type: "error"}
	}
	log.Println("cdnClientIps:", len(cdnClientIps))
	aggData := qos.GetAggData(req, cdnClientIps)
	sortedData := qos.SortData(aggData.CountryCntMap)
	data := qos.PieChartData{
		Title: "用户按国家分布",
		Data:  sortedData,
	}
	return qos.ChartData{Data: data, Type: qos.ChartTypePie}
}

func (u *UsrDistribute) ID() string {
	return "chart_usrDistributionCountry"
}

func (u *UsrDistribute) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    u.ID(),
		Title: "用户按国家分布",
	}
}

package hy

import (
	"log"
	"mikutool/miku/qos"
)

// 用户按大区分布

func init() {
	qos.RegisterChartGenerator(&UsrDistributeArea{})
}

type UsrDistributeArea struct {
}

func (u *UsrDistributeArea) Generate(req qos.QOSRequest) any {
	log.Printf("req: %+v", req)
	cdnClientIps, err := qos.GetHyDistinctCdnClientIps(req)
	if err != nil {
		log.Printf("获取CDN客户端IP失败: %v", err)
		return qos.ChartData{Data: "", Type: "error"}
	}
	log.Println("cdnClientIps:", len(cdnClientIps))
	aggData := qos.GetAggData(req, cdnClientIps)
	sortedData := qos.SortData(aggData.AreaCntMap)
	data := qos.PieChartData{
		Title: "用户按大区分布",
		Data:  sortedData,
	}
	return qos.ChartData{Data: data, Type: qos.ChartTypePie}
}

func (u *UsrDistributeArea) ID() string {
	return "chart_usrDistributionArea"
}

func (u *UsrDistributeArea) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    u.ID(),
		Title: "用户按大区分布图",
	}
}

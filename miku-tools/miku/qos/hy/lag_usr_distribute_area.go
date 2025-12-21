package hy

import (
	"log"
	"mikutool/miku/qos"
)

// 卡顿用户按大区分布

func init() {
	qos.RegisterChartGenerator(&LagUsrDistributeArea{})
}

type LagUsrDistributeArea struct {
}

func (u *LagUsrDistributeArea) Generate(req qos.QOSRequest) any {
	log.Printf("req: %+v", req)
	cdnClientIps, err := qos.GetHyDistinctCdnClientIps(req)
	if err != nil {
		log.Printf("获取CDN客户端IP失败: %v", err)
		return qos.ChartData{Data: "", Type: "error"}
	}
	log.Println("cdnClientIps:", len(cdnClientIps))
	aggData := qos.GetAggData(req, cdnClientIps)
	sortedData := qos.SortData(aggData.AreaLagCntMap)
	data := qos.PieChartData{
		Title: "卡顿用户按大区分布",
		Data:  sortedData,
	}
	return qos.ChartData{Data: data, Type: qos.ChartTypePie}
}

func (u *LagUsrDistributeArea) ID() string {
	return "chart_lagUsrDistributionArea"
}

func (u *LagUsrDistributeArea) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    u.ID(),
		Title: "虎牙卡顿用户按区域分布图",
	}
}

package qos

import (
	"log"
)

// 源站按大区分布饼图

func init() {
	RegisterChartGenerator("upstreamAreaDis", &UpstreamDistribute{})
}

type UpstreamDistribute struct {
}

func (u *UpstreamDistribute) Generate(req QOSRequest) any {
	upstreamDistributeReports, err := GetUpstreamDistributeReport(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	areaCntMap, _ := AggUpstreamDistributeReports(upstreamDistributeReports, req.IpParser)
	sortedAreaData := SortData(areaCntMap)
	data := PieChartData{
		Title: "源站按大区分布",
		Data:  sortedAreaData,
	}
	return data
}

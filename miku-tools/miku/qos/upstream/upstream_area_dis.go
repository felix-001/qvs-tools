package upstream

import (
	"log"
	"mikutool/miku/qos"
)

// 源站按大区分布饼图

func init() {
	qos.RegisterChartGenerator("upstreamAreaDis", &UpstreamDistribute{})
}

type UpstreamDistribute struct {
}

func (u *UpstreamDistribute) Generate(req qos.QOSRequest) any {
	upstreamDistributeReports, err := qos.GetUpstreamDistributeReport(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	areaCntMap, _ := qos.AggUpstreamDistributeReports(upstreamDistributeReports, req.IpParser)
	sortedAreaData := qos.SortData(areaCntMap)
	data := qos.PieChartData{
		Title: "源站按大区分布",
		Data:  sortedAreaData,
	}
	return data
}

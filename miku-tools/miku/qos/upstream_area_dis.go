package qos

import (
	"log"
	"sort"
)

// 源站按大区分布饼图

func init() {
	RegisterChartGenerator("upstreamAreaDis", &UpstreamDistribute{})
}

type UpstreamDistribute struct {
}

func (u *UpstreamDistribute) sortData(data map[string]int) []PieDataItem {
	// 排序数据（按值降序）

	var sortedData []PieDataItem
	for name, value := range data {
		sortedData = append(sortedData, PieDataItem{Name: name, Value: value})
	}

	sort.Slice(sortedData, func(i, j int) bool {
		return sortedData[i].Value > sortedData[j].Value
	})

	// 限制最多显示10个
	if len(sortedData) > 10 {
		sortedData = sortedData[:10]
	}
	return sortedData
}

func (u *UpstreamDistribute) Generate(req QOSRequest) any {
	upstreamDistributeReports, err := GetUpstreamDistributeReport(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	areaCntMap, _ := AggUpstreamDistributeReports(upstreamDistributeReports, req.IpParser)
	sortedAreaData := u.sortData(areaCntMap)
	data := PieChartData{
		Title: "源站按大区分布",
		Data:  sortedAreaData,
	}
	return data

	// 生成源站分布饼图
	//upstreamDistributeChartsHTML := generateUpstreamDistributeChartsHTML(areaCntMap, provCntMap)
	//return upstreamDistributeChartsHTML
}

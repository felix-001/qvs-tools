package hy

import (
	"log"
	"mikutool/miku/qos"
)

// 用户按省分布

func init() {
	qos.RegisterChartGenerator(&UsrDistributeProv{})
}

type UsrDistributeProv struct {
}

func (u *UsrDistributeProv) Generate(req qos.QOSRequest) any {
	log.Printf("req: %+v", req)
	cdnClientIps, err := qos.GetHyDistinctCdnClientIps(req)
	if err != nil {
		log.Printf("获取CDN客户端IP失败: %v", err)
		return qos.ChartData{Data: "", Type: "error"}
	}
	log.Println("cdnClientIps:", len(cdnClientIps))
	aggData := qos.GetAggData(req, cdnClientIps)
	sortedData := qos.SortData(aggData.ProvCntMap)
	data := qos.PieChartData{
		Title: "用户按省分布",
		Data:  sortedData,
	}
	return qos.ChartData{Data: data, Type: qos.ChartTypePie}
}

func (u *UsrDistributeProv) ID() string {
	return "chart_usrDistributionProv"
}

func (u *UsrDistributeProv) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    u.ID(),
		Title: "用户按省份分布图",
	}
}

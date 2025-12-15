package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"

	"github.com/qbox/pili/common/ipdb.v1"
)

func init() {
	RegisterChartGenerator("upstream_dis", &UpstreamDistribute{})
}

type UpstreamDistribute struct {
}

func (u *UpstreamDistribute) Generate(req QOSRequest) any {
	sql := buidUpstreamDistributeSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var upstreamDistributeReports []util.UpstreamDistributeReport
	if err := util.TrinoQuery("miku", sql, &upstreamDistributeReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果upstreamDistributeReports:", len(upstreamDistributeReports))
	areaCntMap, provCntMap := aggUpstreamDistributeReports(upstreamDistributeReports, req.IpParser)

	// 生成源站分布饼图
	upstreamDistributeChartsHTML := generateUpstreamDistributeChartsHTML(areaCntMap, provCntMap)
	return upstreamDistributeChartsHTML
}

func aggUpstreamDistributeReports(reports []util.UpstreamDistributeReport, ipparser *ipdb.City) (map[string]int, map[string]int) {
	areaCntMap := make(map[string]int)
	provCntMap := make(map[string]int)
	for _, report := range reports {
		if report.RemoteAddr == nil {
			continue
		}
		_, _, area, prov := util.GetLocate(*report.RemoteAddr, ipparser)
		areaCntMap[area]++
		provCntMap[prov]++
	}
	return areaCntMap, provCntMap
}

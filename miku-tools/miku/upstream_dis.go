package miku

ype UpstreamDistribute struct {

}

func (u *UpstreamDistribute) Generate(req QOSRequest) string {
	sql = s.buidUpstreamDistributeSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var upstreamDistributeReports []util.UpstreamDistributeReport
	if err := util.TrinoQuery("miku", sql, &upstreamDistributeReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果upstreamDistributeReports:", len(upstreamDistributeReports))
	areaCntMap, provCntMap := s.aggUpstreamDistributeReports(upstreamDistributeReports)

	// 生成源站分布饼图
	upstreamDistributeChartsHTML := s.generateUpstreamDistributeChartsHTML(areaCntMap, provCntMap)	
}

func (s *QOSServer) aggUpstreamDistributeReports(reports []util.UpstreamDistributeReport) (map[string]int, map[string]int) {
	areaCntMap := make(map[string]int)
	provCntMap := make(map[string]int)
	for _, report := range reports {
		if report.RemoteAddr == nil {
			continue
		}
		_, _, area, prov := util.GetLocate(*report.RemoteAddr, s.resources.IpParser)
		areaCntMap[area]++
		provCntMap[prov]++
	}
	return areaCntMap, provCntMap
}
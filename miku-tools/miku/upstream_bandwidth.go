package miku

type UpstreamBandwidth struct {

}

func (u *UpstreamBandwidth) Generate(req QOSRequest) string {
	if req.StreamID == "" || req.FuzzySearch {
		return ""
	}
	var upstreamBandwidthReports []util.UpstreamBandwidthReport
	sql = s.buildMikuUpstreamBandwidthSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &upstreamBandwidthReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果upstreamBandwidthReports:", len(upstreamBandwidthReports))
	// 生成推流/回源带宽折线图
	upstreamBandwidthChartHTML := s.generateUpstreamBandwidthChartHTML(streamUpstreamBandwidthReports)
}
package miku

type struct Fps {

}

func (f *Fps) Generate(req QOSRequest) string {
	if req.StreamID == "" || req.FuzzySearch {
		return ""
	}
	var fpsReports []util.StreamdFpsReport
	sql = s.buildMikuFpsSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &fpsReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果fpsReports:", len(fpsReports))
	// 生成推流/回源帧率折线图
	streamdVideoFpsChartHTML := s.generateStreamdVideoFpsChartHTML(streamdFpsReports)
	streamdAudioFpsChartHTML := s.generateStreamdAudioFpsChartHTML(streamdFpsReports)
}
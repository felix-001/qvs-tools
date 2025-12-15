package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"
)

func init() {
	RegisterChartGenerator("fps", &Fps{})
}

type Fps struct {
}

func (f *Fps) Generate(req QOSRequest) any {
	if !req.Charts.VideoFps {
		return ""
	}
	if req.StreamID == "" || req.FuzzySearch {
		return ""
	}
	var fpsReports []util.StreamdFpsReport
	sql := buildMikuFpsSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &fpsReports); err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果fpsReports:", len(fpsReports))
	// 生成推流/回源帧率折线图
	streamdVideoFpsChartHTML := generateStreamdVideoFpsChartHTML(fpsReports)
	streamdAudioFpsChartHTML := generateStreamdAudioFpsChartHTML(fpsReports)
	return streamdVideoFpsChartHTML + streamdAudioFpsChartHTML
}

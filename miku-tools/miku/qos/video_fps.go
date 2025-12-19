package qos

import (
	"fmt"
	"log"
)

// 视频帧率趋势图

func init() {
	RegisterChartGenerator("videoFps", &VideoFps{})
}

type VideoFps struct {
}

func (f *VideoFps) Generate(req QOSRequest) any {
	if req.StreamID == "" || req.FuzzySearch {
		return ""
	}
	reports, err := GetStreamedFpsReport(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果fpsReports:", len(reports))
	fn := func(cb Cb) {
		for _, report := range reports {
			if report.Avg_IncomingVideoFps == nil {
				continue
			}
			cb(*report.Ts, *report.Avg_IncomingVideoFps)
		}
	}
	chartData := GenerateLineChartData("视频帧率趋势图", "帧率", fn)
	return chartData
}

/*
	// 生成推流/回源帧率折线图
	streamdVideoFpsChartHTML := generateStreamdVideoFpsChartHTML(fpsReports)
	streamdAudioFpsChartHTML := generateStreamdAudioFpsChartHTML(fpsReports)
	return streamdVideoFpsChartHTML + streamdAudioFpsChartHTML
}
*/

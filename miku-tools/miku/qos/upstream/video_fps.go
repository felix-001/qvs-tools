package upstream

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 视频帧率趋势图

func init() {
	qos.RegisterChartGenerator("videoFps", &VideoFps{})
}

type VideoFps struct {
}

func (f *VideoFps) Generate(req qos.QOSRequest) any {
	if req.StreamID == "" || req.FuzzySearch {
		return ""
	}
	reports, err := qos.GetStreamedFpsReport(req)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果fpsReports:", len(reports))
	fn := func(cb qos.Cb) {
		for _, report := range reports {
			if report.Avg_IncomingVideoFps == nil {
				continue
			}
			cb(*report.Ts, *report.Avg_IncomingVideoFps)
		}
	}
	chartData := qos.GenerateLineChartData("视频帧率趋势图", "帧率", fn)
	return chartData
}

/*
	// 生成推流/回源帧率折线图
	streamdVideoFpsChartHTML := generateStreamdVideoFpsChartHTML(fpsReports)
	streamdAudioFpsChartHTML := generateStreamdAudioFpsChartHTML(fpsReports)
	return streamdVideoFpsChartHTML + streamdAudioFpsChartHTML
}
*/

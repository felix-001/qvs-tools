package qos

import (
	"fmt"
	"log"
)

// 音频帧率趋势图

func init() {
	RegisterChartGenerator("audioFps", &AudioFps{})
}

type AudioFps struct {
}

func (f *AudioFps) Generate(req QOSRequest) any {
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
			if report.Avg_IncomingAudioFps == nil {
				continue
			}
			cb(*report.Ts, *report.Avg_IncomingAudioFps)
		}
	}
	chartData := GenerateLineChartData("音频帧率趋势图", "帧率", fn)
	return chartData
}

/*
	// 生成推流/回源帧率折线图
	streamdVideoFpsChartHTML := generateStreamdVideoFpsChartHTML(fpsReports)
	streamdAudioFpsChartHTML := generateStreamdAudioFpsChartHTML(fpsReports)
	return streamdVideoFpsChartHTML + streamdAudioFpsChartHTML
}
*/

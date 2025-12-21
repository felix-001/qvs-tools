package upstream

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
)

// 音频帧率趋势图

func init() {
	qos.RegisterChartGenerator(&AudioFps{})
}

type AudioFps struct {
}

func (f *AudioFps) ID() string {
	return "chart_audioFps"
}

func (f *AudioFps) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    f.ID(),
		Title: "音频帧率趋势图",
	}
}

func (f *AudioFps) Generate(req qos.QOSRequest) any {
	if req.StreamID == "" || req.FuzzySearch {
		return qos.ChartData{Data: "", Type: "error"}
	}
	reports, err := qos.GetStreamedFpsReport(req)
	if err != nil {
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果fpsReports:", len(reports))
	fn := func(cb qos.Cb) {
		for _, report := range reports {
			if report.Avg_IncomingAudioFps == nil {
				continue
			}
			cb(*report.Ts, *report.Avg_IncomingAudioFps)
		}
	}
	chartData := qos.GenerateLineChartData("音频帧率趋势图", "帧率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

/*
	// 生成推流/回源帧率折线图
	streamdVideoFpsChartHTML := generateStreamdVideoFpsChartHTML(fpsReports)
	streamdAudioFpsChartHTML := generateStreamdAudioFpsChartHTML(fpsReports)
	return streamdVideoFpsChartHTML + streamdAudioFpsChartHTML
}
*/

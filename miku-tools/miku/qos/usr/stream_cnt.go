package usr

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
	"strings"
)

func init() {
	qos.RegisterChartGenerator(&StreamCnt{})
}

type StreamCnt struct {
}

func (s *StreamCnt) ID() string {
	return "chart_onlineStreams"
}

func (s *StreamCnt) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    s.ID(),
		Title: "在线流个数趋势图",
	}
}

func (s *StreamCnt) Generate(req qos.QOSRequest) any {
	sql := qos.BuildMikuStreamCntSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}

	var streamCntReports []util.StreamdStreamCntReport
	if err := util.TrinoQuery("miku", sql, &streamCntReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果streamCntReports:", len(streamCntReports))
	data := qos.LineChartData{
		Title:       "在线流个数趋势图",
		SeriesTitle: "流个数",
		Color:       "#1890ff",
		XType:       "category",
		XAxis:       []string{},
		YAxis:       []string{},
	}
	for _, report := range streamCntReports {
		if report.StreamCnt == nil {
			continue
		}
		ts := strings.ReplaceAll(*report.Ts_m, "+08:00", "")
		// Remove year from timestamp (format: 2025-12-16T12:49:00)
		if len(ts) >= 10 {
			ts = ts[5:] // Keep everything after the year
		}
		data.XAxis = append(data.XAxis, ts)
		data.YAxis = append(data.YAxis, fmt.Sprintf("%d", *report.StreamCnt))
	}
	return qos.ChartData{Data: data, Type: qos.ChartTypeLine}
}

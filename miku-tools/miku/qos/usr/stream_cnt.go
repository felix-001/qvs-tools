package usr

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
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
	fn := func(cb qos.Cb) {
		for _, report := range streamCntReports {
			if report.StreamCnt == nil {
				continue
			}
			cb(*report.Ts_m, *report.StreamCnt)
		}
	}
	chartData := qos.GenerateLineChartData("在线流个数趋势图", "流个数", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

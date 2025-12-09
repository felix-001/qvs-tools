package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"
)

func init() {
	RegisterChartGenerator("stream_cnt", &StreamCnt{})
}

type StreamCnt struct {
}

func (s *StreamCnt) Generate(req QOSRequest) string {
	if !req.Charts.OnlineStreams {
		return ""
	}
	sql := buildMikuStreamCntSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}

	var streamCntReports []util.StreamdStreamCntReport
	if err := util.TrinoQuery("miku", sql, &streamCntReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return fmt.Sprintf("查询失败: %v", err)
	}
	log.Println("查询结果streamCntReports:", len(streamCntReports))

	// 生成在线流个数折线图
	streamCntChartHTML := generateStreamCntChartHTML(streamCntReports)
	return streamCntChartHTML
}

package qos

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/public/util"
	"strings"
)

type ChartMgr struct {
	charts   []ChartConf
	chartMap map[string]ChartConf
}

func NewChartMgr() *ChartMgr {
	return &ChartMgr{
		chartMap: make(map[string]ChartConf),
	}
}

func (c *ChartMgr) Parse() error {
	charts := make([]ChartConf, 0)
	if err := json.Unmarshal([]byte(Charts_conf_json), &charts); err != nil {
		log.Printf("parse json fail: %v\n", err)
		return err
	}
	log.Printf("charts: %+v\n", charts)
	c.charts = charts
	for _, chart := range charts {
		c.chartMap[chart.Name] = chart
	}
	log.Printf("chart map loaded: %+v\n", c.chartMap)
	return nil
}

func (c *ChartConf) buildMikuWhere(req QOSRequest) (string, error) {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)
	where := c.SQL.Where
	where += fmt.Sprintf("AND from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'\n", req.StartTime, req.EndTime)
	where += fmt.Sprintf("AND day >= '%s' and day <= '%s'\n", startDay, endDay)
	if req.StreamID != "" {
		if req.FuzzySearch {
			where += fmt.Sprintf("AND StreamName LIKE '%%%s%%'\n", req.StreamID)
		} else {
			where += fmt.Sprintf("AND StreamName = '%s'\n", req.StreamID)
		}
	}
	if req.AppName != "" {
		where += fmt.Sprintf("AND AppName = '%s'\n", req.AppName)
	}
	if req.CdnIp != "" {
		where += fmt.Sprintf("AND remoteaddr like '%%%s%%'\n", req.CdnIp)
	}
	if req.Domain != "" {
		where += fmt.Sprintf("AND Domain = '%s'\n", req.Domain)
	}
	if req.Protocol != "" {
		where += fmt.Sprintf("AND Protocol = '%s'\n", req.Protocol)
	}
	// 添加剔除流ID过滤
	if req.ExcludeStreams != "" {
		// 将逗号分隔的流ID列表转换为SQL NOT IN条件
		excludeStreams := strings.Split(req.ExcludeStreams, ",")
		for _, stream := range excludeStreams {
			stream = strings.TrimSpace(stream)
			if req.FuzzySearch {
				where += fmt.Sprintf("AND StreamName not like '%%%s%%'\n", stream)
			} else {
				where += fmt.Sprintf("AND StreamName != '%s'\n", stream)
			}
		}
	}
	return where, nil
}

func (c *ChartConf) buildHyWhereCommonPart(req QOSRequest) string {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

	where := "\tAND " + c.SQL.Where
	where += fmt.Sprintf("\tAND from_unixtime(cts) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'\n", req.StartTime, req.EndTime)
	where += fmt.Sprintf("\tAND day >= '%s' and day <= '%s'\n", startDay, endDay)
	where += "\tAND dim_heart_type != '0'\n"
	where += "\tAND dim_stream_url not like '%.pstream%'\n"
	where += "\tAND (client_type = 'sdk_video_bad_quality_ratio' OR client_type = 'web_video_bad_quality_ratio')\n"
	return where
}

func (c *ChartConf) buildHyWhere(req QOSRequest) (string, error) {
	where := c.buildHyWhereCommonPart(req)
	if req.StreamID != "" {
		streamID := strings.ToLower(req.StreamID)
		where += fmt.Sprintf(`\tAND dim_stream_url like '%%%s%%'\n`, streamID)
	}

	if req.Domain != "" {
		where += fmt.Sprintf("\tAND dim_stream_url like '%%%s%%'\n", req.Domain)
	}

	if req.UserIp != "" {
		where += fmt.Sprintf(`\tAND dim__ip = '%s'\n`, req.UserIp)
	}

	if req.CdnIp != "" {
		where += fmt.Sprintf(`\tAND dim_cdnip = '%s'\n`, req.CdnIp)
	}

	// 添加剔除流ID过滤
	if req.ExcludeStreams != "" {
		// 将逗号分隔的流ID列表转换为SQL NOT IN条件
		excludeStreams := strings.Split(req.ExcludeStreams, ",")
		for _, stream := range excludeStreams {
			stream = strings.TrimSpace(stream)
			stream = strings.ToLower(stream)
			where += fmt.Sprintf(`\tAND dim_stream_url not like '%%%s%%'`, stream)
		}
	}

	switch req.Protocol {
	case "hls":
	case "p2p":
		where += `\tAND dim_p2p = '1'\n`
	case "flv":
		where += `\tAND dim_p2p = '0'\n`
	}
	return where, nil
}

func (c *ChartConf) buildWhere(req QOSRequest) (string, error) {
	switch c.Table {
	case "hy":
		return c.buildHyWhere(req)
	case "miku":
		return c.buildMikuWhere(req)
	default:
		return "", fmt.Errorf("不支持的表: %s", c.SQL.From)
	}
}

func (c *ChartConf) buildSelect(req QOSRequest) (string, error) {
	choose := c.SQL.Select
	switch c.Table {
	case "hy":
		if c.SQL.GroupByMinute {
			choose += ", date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai') as ts_m"
		}
	case "miku":
	default:
		return "", fmt.Errorf("不支持的表: %s", c.Table)
	}
	return choose, nil
}

func (c *ChartConf) buildSql(req QOSRequest) (string, error) {
	var sql string
	if c.SQL.With != "" {
		sql = fmt.Sprintf("WITH\n\t%s\n", c.SQL.With)
	}

	choose, err := c.buildSelect(req)
	if err != nil {
		return "", err
	}
	where, err := c.buildWhere(req)
	if err != nil {
		return "", err
	}

	sql += fmt.Sprintf(`
SELECT 
	%s 
FROM 
	%s
WHERE 1=1
	%s
`,
		choose, c.SQL.From, where)

	if c.SQL.GroupBy != "" {
		sql += fmt.Sprintf("GROUP BY\n\t%s\n", c.SQL.GroupBy)
		if c.SQL.GroupByMinute {
			sql += ", date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')"
		}
	}
	if c.SQL.OrderBy != "" {
		sql += fmt.Sprintf("\nORDER BY\n\t%s\n", c.SQL.OrderBy)
	}
	if req.LogLevel == "detail" {
		log.Println("sql:\n", sql)
	}
	return sql, nil
}

type LineData struct {
	XAxis []any
	YAxis []any
}

func (c *ChartConf) getLineData(results []map[string]any) any {
	if c.SQL.Dimension != "" {
		datas := make(map[string]*LineData)
		for _, result := range results {
			dimensionValue := result[c.SQL.Dimension].(string)
			if _, ok := datas[dimensionValue]; !ok {
				datas[dimensionValue] = &LineData{}
			}
			datas[dimensionValue].YAxis = append(datas[dimensionValue].YAxis, result[c.SQL.Field])
			datas[dimensionValue].XAxis = append(datas[dimensionValue].XAxis, result["ts_m"])
		}
		return datas
	} else {
		data := &LineData{}
		for _, result := range results {
			data.YAxis = append(data.YAxis, result[c.SQL.Field])
			data.XAxis = append(data.XAxis, result["ts_m"])
		}
		return data
	}
}

func (c *ChartConf) Query(req QOSRequest) (any, error) {
	sql, err := c.buildSql(req)
	if err != nil {
		log.Println("build sql error:", err)
		return "", err
	}
	var results []map[string]any
	if err := util.TrinoQueryMap("miku", sql, &results); err != nil {
		return nil, fmt.Errorf("query err: %v", err)
	}
	//log.Printf("results: %+v\n", results)
	switch c.Type {
	case "line":
		return c.getLineData(results), nil
	case "table":
	case "pie":
	}

}

func (c *ChartConf) GetChartInfo() ChartInfo {
	return ChartInfo{
		ID:    c.Name,
		Title: c.Title,
	}
}

func (c *ChartMgr) GetChartInfos() []ChartInfo {
	infos := make([]ChartInfo, 0, len(c.charts))
	for _, chart := range c.charts {
		infos = append(infos, chart.GetChartInfo())
	}
	return infos
}

func (c *ChartMgr) Query(req QOSRequest) (any, error) {
	log.Println("Query chart:", req.Chart)
	chart, ok := c.chartMap[req.Chart]
	if !ok {
		return nil, fmt.Errorf("chart not found: %s", req.Chart)
	}
	return chart.Query(req)
}

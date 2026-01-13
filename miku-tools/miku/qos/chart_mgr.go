package qos

import (
	"crypto/md5"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"
	"strings"
	"time"
)

type ChartMgr struct {
	charts   []config.ChartConf
	chartMap map[string]config.ChartConf
}

func NewChartMgr() *ChartMgr {
	return &ChartMgr{
		chartMap: make(map[string]config.ChartConf),
	}
}

func (c *ChartMgr) Parse(conf *config.Config) error {
	c.charts = conf.ChartConfigs
	for _, chart := range conf.ChartConfigs {
		c.chartMap[chart.Name] = chart
	}
	log.Printf("chart map loaded: %+v\n", c.chartMap)
	return nil
}

func (c *ChartMgr) buildMikuWhere(req QOSRequest, chartConf *config.ChartConf) (string, error) {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)
	where := chartConf.SQL.Where
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

func (c *ChartMgr) buildHyWhereCommonPart(req QOSRequest, chartConf *config.ChartConf) string {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

	where := "\tAND " + chartConf.SQL.Where
	where += fmt.Sprintf("\tAND from_unixtime(cts) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'\n", req.StartTime, req.EndTime)
	where += fmt.Sprintf("\tAND day >= '%s' and day <= '%s'\n", startDay, endDay)
	where += "\tAND dim_heart_type != '0'\n"
	where += "\tAND dim_stream_url not like '%.pstream%'\n"
	where += "\tAND (client_type = 'sdk_video_bad_quality_ratio' OR client_type = 'web_video_bad_quality_ratio')\n"
	return where
}

func (c *ChartMgr) buildHyWhere(req QOSRequest, chartConf *config.ChartConf) (string, error) {
	where := c.buildHyWhereCommonPart(req, chartConf)
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

func (c *ChartMgr) buildWhere(req QOSRequest, chartConf *config.ChartConf) (string, error) {
	switch chartConf.Table {
	case "hy":
		return c.buildHyWhere(req, chartConf)
	case "miku":
		return c.buildMikuWhere(req, chartConf)
	default:
		return "", fmt.Errorf("不支持的表: %s", chartConf.SQL.From)
	}
}

func (c *ChartMgr) buildSelect(req QOSRequest, chartConf *config.ChartConf) (string, error) {
	choose := chartConf.SQL.Select
	switch chartConf.Table {
	case "hy":
		if chartConf.SQL.GroupByMinute {
			choose += ", date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai') as ts_m"
		}
	case "miku":
	default:
		return "", fmt.Errorf("不支持的表: %s", chartConf.Table)
	}
	return choose, nil
}

func (c *ChartMgr) buildSql(req QOSRequest, chartConf *config.ChartConf) (string, error) {
	var sql string
	if chartConf.SQL.With != "" {
		sql = fmt.Sprintf("WITH\n\t%s\n", chartConf.SQL.With)
	}

	choose, err := c.buildSelect(req, chartConf)
	if err != nil {
		return "", err
	}
	where, err := c.buildWhere(req, chartConf)
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
		choose, chartConf.SQL.From, where)

	if chartConf.SQL.GroupBy != "" {
		sql += fmt.Sprintf("GROUP BY\n\t%s\n", chartConf.SQL.GroupBy)
		if chartConf.SQL.GroupByMinute {
			sql += ", date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')"
		}
	}
	if chartConf.SQL.OrderBy != "" {
		sql += fmt.Sprintf("\nORDER BY\n\t%s\n", chartConf.SQL.OrderBy)
	}
	if req.LogLevel == "detail" {
		log.Println("sql:\n", sql)
	}
	return sql, nil
}

func (c *ChartMgr) getLineData(results []map[string]any, chartConf *config.ChartConf) any {
	seriesData := map[string]*SeriesData{}
	data := LineChartData2{
		XAxis:      []string{},
		XType:      "category",
		SeriesData: seriesData,
	}
	lastTs := ""
	for _, result := range results {
		dimensionValue := chartConf.SeriesTitle
		if chartConf.SQL.Dimension != "" {
			if _, ok := result[chartConf.SQL.Dimension]; !ok {
				log.Printf("err, dimension nil, result: %+v\n", result)
				continue
			}
			if result[chartConf.SQL.Dimension] == nil {
				log.Printf("err, dimension nil, result: %+v\n", result)
				continue
			}
			dimensionValue = result[chartConf.SQL.Dimension].(string)
		}
		if _, ok := seriesData[dimensionValue]; !ok {
			seriesData[dimensionValue] = &SeriesData{}
		}
		seriesData[dimensionValue].YAxis = append(seriesData[dimensionValue].YAxis, result[chartConf.SQL.Field].(string))
		t := result["ts_m"].(time.Time).Format("2006-01-02 15:04:05")
		if lastTs == "" || t != lastTs {
			data.XAxis = append(data.XAxis, t)
			lastTs = t
		}

	}
	chartData := ChartData{
		Type:  chartConf.Type,
		Data:  data,
		Title: chartConf.Title,
	}
	return chartData
}

type ResultCache struct {
	Cache []map[string]any
	Req   QOSRequest
}

var resultCache map[string]ResultCache

func (c *ChartMgr) needRefresh(req QOSRequest, md5 string) bool {
	cache, ok := resultCache[md5]
	if !ok {
		return true
	}
	cacheReq := cache.Req
	if req.StartTime != cacheReq.StartTime ||
		req.EndTime != cacheReq.EndTime ||
		req.RequestId != cacheReq.RequestId ||
		req.Protocol != cacheReq.Protocol ||
		req.StreamID != cacheReq.StreamID ||
		req.Domain != cacheReq.Domain ||
		req.UserIp != cacheReq.UserIp ||
		req.CdnIp != cacheReq.CdnIp ||
		req.ExcludeStreams != cacheReq.ExcludeStreams ||
		len(cache.Cache) == 0 {
		return true
	}
	return false
}

func (c *ChartMgr) DoQuery(req QOSRequest, chartConf *config.ChartConf) (any, error) {
	sql, err := c.buildSql(req, chartConf)
	if err != nil {
		log.Println("build sql error:", err)
		return "", err
	}

	sqlHash := fmt.Sprintf("%x", md5.Sum([]byte(sql)))
	log.Printf("SQL MD5 hash: %s", sqlHash)

	var results []map[string]any
	if c.needRefresh(req, sqlHash) {
		if err := util.TrinoQueryMap("miku", sql, &results); err != nil {
			return nil, fmt.Errorf("query err: %v", err)
		}
		resultCache[sqlHash] = ResultCache{
			Req:   req,
			Cache: results,
		}
	} else {
		results = resultCache[sqlHash].Cache
	}
	//log.Printf("results: %+v\n", results)
	switch chartConf.Type {
	case "line":
		return c.getLineData(results, chartConf), nil
	case "table":
	case "pie":
		//return c.getPieData(results), nil
	}
	return "", nil
}

func (c *ChartMgr) GetChartInfo(chartConf *config.ChartConf) ChartInfo {
	return ChartInfo{
		ID:    chartConf.Name,
		Title: chartConf.Title,
	}
}

func (c *ChartMgr) GetChartInfos() []ChartInfo {
	infos := make([]ChartInfo, 0, len(c.charts))
	for _, chart := range c.charts {
		infos = append(infos, c.GetChartInfo(&chart))
	}
	return infos
}

func (c *ChartMgr) Query(req QOSRequest) (any, error) {
	log.Println("Query chart:", req.Chart)
	chart, ok := c.chartMap[req.Chart]
	if !ok {
		return nil, fmt.Errorf("chart not found: %s", req.Chart)
	}
	return c.DoQuery(req, &chart)
}

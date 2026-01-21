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

func (c *ChartMgr) buildMikuWhere(req QOSRequest, chartConf *config.SingleSQL) (string, error) {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)
	where := ""
	if chartConf.Where != "" {
		where += "\tAND " + chartConf.Where
	}
	where += fmt.Sprintf(" AND from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'\n", req.StartTime, req.EndTime)
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

func (c *ChartMgr) buildHyWhereCommonPart(req QOSRequest, chartConf *config.SingleSQL) string {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

	where := ""
	if chartConf.Where != "" {
		where += "\tAND " + chartConf.Where
	}
	where += fmt.Sprintf("\tAND from_unixtime(cts) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'\n", req.StartTime, req.EndTime)
	where += fmt.Sprintf("\tAND day >= '%s' and day <= '%s'\n", startDay, endDay)
	where += "\tAND dim_heart_type != '0'\n"
	where += "\tAND dim_stream_url not like '%.pstream%'\n"
	where += "\tAND (client_type = 'sdk_video_bad_quality_ratio' OR client_type = 'web_video_bad_quality_ratio')\n"
	return where
}

func (c *ChartMgr) buildHyWhere(req QOSRequest, chartConf *config.SingleSQL) (string, error) {
	where := c.buildHyWhereCommonPart(req, chartConf)
	if req.StreamID != "" {
		streamID := strings.ToLower(req.StreamID)
		where += fmt.Sprintf("\tAND dim_stream_url like '%%%s%%'\n", streamID)
	}

	if req.Domain != "" {
		where += fmt.Sprintf("\tAND dim_stream_url like '%%%s%%'\n", req.Domain)
	}

	if req.UserIp != "" {
		where += fmt.Sprintf("\tAND dim__ip = '%s'\n", req.UserIp)
	}

	if req.CdnIp != "" {
		where += fmt.Sprintf("\tAND dim_cdnip = '%s'\n", req.CdnIp)
	}

	// 添加剔除流ID过滤
	if req.ExcludeStreams != "" {
		// 将逗号分隔的流ID列表转换为SQL NOT IN条件
		excludeStreams := strings.Split(req.ExcludeStreams, ",")
		for _, stream := range excludeStreams {
			stream = strings.TrimSpace(stream)
			stream = strings.ToLower(stream)
			where += fmt.Sprintf("\tAND dim_stream_url not like '%%%s%%'\n", stream)
		}
	}

	switch req.Protocol {
	case "hls":
	case "p2p":
		where += "\tAND dim_p2p = '1'\n"
	case "flv":
		where += "\tAND dim_p2p = '0'\n"
	}
	return where, nil
}

func (c *ChartMgr) buildWhere(req QOSRequest, chartConf *config.SingleSQL) (string, error) {
	switch chartConf.Table {
	case "hy":
		return c.buildHyWhere(req, chartConf)
	case "miku":
		return c.buildMikuWhere(req, chartConf)
	default:
		return "", fmt.Errorf("buildWhere, 不支持的表: %s, %+v", chartConf.Table, chartConf)
	}
}

func (c *ChartMgr) buildSelect(req QOSRequest, chartConf *config.SingleSQL) (string, error) {
	choose := chartConf.Select
	switch chartConf.Table {
	case "hy":
		if chartConf.GroupByMinute {
			choose += ", date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai') as ts_m"
		}
	case "miku":
		if chartConf.GroupByMinute {
			choose += ", date_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai') as ts_m"
		}
	default:
		return "", fmt.Errorf("buildSelect 不支持的表: %s, %+v", chartConf.Table, chartConf)
	}
	return choose, nil
}

func (c *ChartMgr) buildWith(req QOSRequest, chartConf *config.ChartConf) (string, error) {
	with := "WITH\n\t"
	for _, conf := range chartConf.SQL.With {
		//log.Printf("with conf: %+v\n", conf)
		//log.Printf("with sql: %+v\n", conf.SingleSQL)
		sql, err := c.buildSql(req, &conf.SingleSQL)
		if err != nil {
			return "", err
		}
		with += fmt.Sprintf("%s AS (%s),\n\t", conf.Name, sql)
	}
	with = with[:len(with)-3] + "\n"
	return with, nil
}

func (c *ChartMgr) buildSql(req QOSRequest, chartConf *config.SingleSQL) (string, error) {
	var sql string

	if chartConf.Raw != "" {
		return chartConf.Raw, nil
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
		choose, chartConf.From, where)

	if chartConf.GroupBy != "" {
		sql += fmt.Sprintf("GROUP BY\n\t%s\n", chartConf.GroupBy)
		if chartConf.GroupByMinute {
			if chartConf.Table == "hy" {
				sql += ", date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')"
			} else {
				sql += ", date_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai')"
			}
		}
	} else if chartConf.GroupByMinute {
		if chartConf.Table == "hy" {
			sql += "GROUP BY\n\tdate_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')"
		} else {
			sql += "GROUP BY\n\tdate_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai')"
		}
	}
	if chartConf.OrderBy != "" {
		sql += fmt.Sprintf("\nORDER BY\n\t%s\n", chartConf.OrderBy)
	}
	return sql, nil
}

func (c *ChartMgr) buildFinalSQL(req QOSRequest, chartConf *config.ChartConf) (string, error) {
	with := ""
	log.Printf("with: %+v\n", chartConf.SQL.With)
	if len(chartConf.SQL.With) != 0 {
		var err error
		with, err = c.buildWith(req, chartConf)
		if err != nil {
			return "", err
		}
	}
	if req.LogLevel == "detail" {
		log.Printf("with SQL result: %s\n", with)
	}

	sql, err := c.buildSql(req, chartConf.SQL.Final)
	if err != nil {
		log.Printf("buildSql final err:%v, final: %+v\n", err, chartConf.SQL.Final)
		return "", err
	}

	if req.LogLevel == "detail" {
		log.Println("sql:\n", with+sql)
	}
	return with + sql, nil
}

func (c *ChartMgr) getDimensionValue(req QOSRequest, field string, result map[string]any, chartConf *config.ChartConf) string {
	dimensionValue := chartConf.SeriesTitle
	if chartConf.SQL.Dimension != "" {
		if _, ok := result[chartConf.SQL.Dimension]; !ok {
			log.Printf("err, dimension nil, result: %+v\n", result)
			return ""
		}
		if result[chartConf.SQL.Dimension] == nil {
			log.Printf("err, dimension nil, result: %+v\n", result)
			return ""
		}
		dimensionValue = result[chartConf.SQL.Dimension].(string)
	}
	dimensionValue += "_" + field
	if req.Protocol != "" {
		dimensionValue += "_" + req.Protocol
	}
	if req.Domain != "" {
		dimensionValue += "_" + req.Domain
	}
	if req.UserIp != "" {
		dimensionValue += "_usr:" + req.UserIp
	}
	if req.CdnIp != "" {
		dimensionValue += "_cdn:" + req.CdnIp
	}
	if req.StreamID != "" {
		dimensionValue += "_" + req.StreamID
	}

	return dimensionValue
}

func (c *ChartMgr) getLineData(req QOSRequest, results []map[string]any, chartConf *config.ChartConf) any {
	seriesData := map[string]*SeriesData{}
	data := LineChartData2{
		XAxis:      []string{},
		XType:      "category",
		SeriesData: seriesData,
	}
	lastTs := ""
	for _, result := range results {
		for _, field := range chartConf.SQL.Fields {
			dimensionValue := c.getDimensionValue(req, field, result, chartConf)
			if dimensionValue == "" {
				log.Println("err, dimensionValue or field is empty", "dimensionValue:", dimensionValue, "field:", field)
				continue
			}
			if _, ok := seriesData[dimensionValue]; !ok {
				seriesData[dimensionValue] = &SeriesData{}
			}
			value, ok := result[field].(string)
			if !ok {
				value_int, ok := result[field].(int64)
				if !ok {
					value_float := result[field].(float64)
					value = fmt.Sprintf("%.1f", value_float)
				} else {
					value = fmt.Sprintf("%d", value_int)
				}
			}
			seriesData[dimensionValue].YAxis = append(seriesData[dimensionValue].YAxis, value)
			if result["ts_m"] != nil {
				t := result["ts_m"].(time.Time).Format("2006-01-02 15:04:05")
				if lastTs == "" || t != lastTs {
					data.XAxis = append(data.XAxis, t)
					lastTs = t
				}
			}
			if result["ts"] != nil {
				t := result["ts"].(time.Time).Format("2006-01-02 15:04:05")
				if lastTs == "" || t != lastTs {
					data.XAxis = append(data.XAxis, t)
					lastTs = t
				}
			}
		}

	}
	chartData := ChartData{
		Type:  chartConf.Type,
		Data:  data,
		Title: chartConf.Title,
	}
	return chartData
}
func (c *ChartMgr) getPieData(req QOSRequest, results []map[string]any, chartConf *config.ChartConf) (ChartData, error) {
	items := []PieDataItem{}
	for _, result := range results {
		if result[chartConf.SQL.PieName] == nil {
			log.Println("err, pieName nil, result:", result)
			continue
		}
		if result[chartConf.SQL.PieValue] == nil {
			log.Println("err, pieValue nil, result:", result)
			continue
		}
		item := PieDataItem{
			Name:  result[chartConf.SQL.PieName].(string),
			Value: int(result[chartConf.SQL.PieValue].(int64)),
		}
		items = append(items, item)
	}
	pieData := PieChartData{
		Title: chartConf.Title,
		Data:  items,
	}
	chartData := ChartData{
		Type:  ChartTypePie,
		Data:  pieData,
		Title: chartConf.Title,
	}
	return chartData, nil
}

func (c *ChartMgr) getTableData(req QOSRequest, results []map[string]any, chartConf *config.ChartConf) (ChartData, error) {
	if req.LogLevel == "detail" {
		log.Printf("%+v\n", results)
	}
	tableData := map[string]any{
		"title": chartConf.Title,
		"data":  results,
	}
	chartData := ChartData{
		Type:  ChartTypeTable,
		Data:  tableData,
		Title: chartConf.Title,
	}
	return chartData, nil
}

type ResultCache struct {
	Cache []map[string]any
	Req   QOSRequest
}

var resultCache map[string]ResultCache = make(map[string]ResultCache)

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
	sql, err := c.buildFinalSQL(req, chartConf)
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
	if req.LogLevel == "detail" {
		log.Printf("results: %+v\n", results)
	}
	switch chartConf.Type {
	case "line":
		return c.getLineData(req, results, chartConf), nil
	case "table":
		return c.getTableData(req, results, chartConf)
	case "pie":
		return c.getPieData(req, results, chartConf)
	}
	return "", nil
}

func (c *ChartMgr) GetChartInfo(chartConf *config.ChartConf) ChartInfo {
	return ChartInfo{
		ID:    chartConf.Name,
		Title: chartConf.Title,
		Table: chartConf.Table,
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

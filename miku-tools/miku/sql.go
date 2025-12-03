package miku

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// buildSQLQuery 构建SQL查询语句
func (s *QOSServer) buildSQLQuery(req QOSRequest, raw bool) string {

	streamId := strings.ToLower(req.StreamID)
	// 基础查询
	sql := "SELECT " //"* FROM huyabiz_quality_report_log WHERE 1=1 "
	if raw {
		sql += "*"
	} else {
		sql += "date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')  as ts, " +
			"COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent," +
			"COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt," +
			"COUNT(*) as total"
	}
	sql += " FROM huyabiz_quality_report_log WHERE 1=1 "

	// 添加时间范围过滤（转换为day格式）
	if req.StartTime != "" {
		startDay := s.convertToDay(req.StartTime)
		log.Printf("StartTime输入: %s, 转换后: %s", req.StartTime, startDay)
		sql += fmt.Sprintf(" AND day >= '%s'", startDay)
	}

	if req.EndTime != "" {
		endDay := s.convertToDay(req.EndTime)
		log.Printf("EndTime输入: %s, 转换后: %s", req.EndTime, endDay)
		sql += fmt.Sprintf(" AND day <= '%s'", endDay)
	}

	// 添加流ID过滤
	if req.StreamID != "" {
		if req.FuzzySearch {
			sql += fmt.Sprintf(" AND dim_stream_url LIKE '%%%s%%'", streamId)
		} else {
			sql += fmt.Sprintf(" AND dim_stream = '%s'", streamId)
		}
	}

	// 添加域名过滤
	if req.Domain != "" {
		if req.FuzzySearch {
			sql += fmt.Sprintf(" AND dim_stream_url LIKE '%%%s%%'", req.Domain)
		} else {
			sql += fmt.Sprintf(" AND dim_cdndomain = '%s'", req.Domain)
		}
	}

	// 添加UID过滤（假设UID在stream_url中，如果没有相应字段可以注释掉）
	if req.UID != "" {
		sql += fmt.Sprintf(" AND dim_stream_url LIKE '%%%s%%'", req.UID)
	}

	// 添加小时过滤
	if req.Hour != "" {
		// 验证小时格式
		if hour, err := strconv.Atoi(req.Hour); err == nil && hour >= 0 && hour <= 23 {
			sql += fmt.Sprintf(" AND hour = '%d'", hour)
		} else {
			log.Printf("无效的小时格式: %s，已忽略小时过滤", req.Hour)
		}
	}

	sql += " AND (client_type = 'sdk_video_bad_quality_ratio' or client_type = 'web_video_bad_quality_ratio')"
	if !raw {
		sql += " GROUP BY date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')" +
			" ORDER BY ts"
	}

	// 限制结果数量
	sql += " LIMIT 10000"

	return sql
}

func (s *QOSServer) buildMikuSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := s.convertToDay(req.StartTime)
	endDay := s.convertToDay(req.EndTime)

	//return "select * from dwd_flowd_miku_streamd_log where day = '20251203' limit 10"
	sql := fmt.Sprintf(`
        SELECT date_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai') as ts_m,

		SUM(IF(type = 'player', lagduration, 0)) AS total_lag_duration_player,
		SUM(IF(type = 'player', costtime, 0)) AS total_cost_time_player,
		ROUND(SUM(IF(type = 'player', lagduration, 0)) * 100 / NULLIF(SUM(IF(type = 'player', costtime, 0)), 0), 4) AS ratio_lag_player,

		SUM(IF(type = 'player' AND hitinfo = 'TCP_HIT -', lagduration, 0)) AS hit_total_lag_duration_player,
		SUM(IF(type = 'player' AND hitinfo = 'TCP_HIT -', costtime, 0)) AS hit_total_cost_time_player,
		ROUND(SUM(IF(type = 'player' AND hitinfo = 'TCP_HIT -', lagduration, 0)) / NULLIF(SUM(IF(type = 'player' AND hitinfo = 'TCP_HIT -', costtime, 0)), 0), 4) AS hit_ratio_lag_player,

		SUM(IF(type = 'player' AND hitinfo != 'TCP_HIT -', lagduration, 0)) AS not_hit_total_lag_duration_player,
		SUM(IF(type = 'player' AND hitinfo != 'TCP_HIT -', costtime, 0)) AS not_hit_total_cost_time_player,
		ROUND(SUM(IF(type = 'player' AND hitinfo != 'TCP_HIT -', lagduration, 0)) / NULLIF(SUM(IF(type = 'player' AND hitinfo != 'TCP_HIT -', costtime, 0)), 0), 4) AS not_hit_ratio_lag_player,

		SUM(IF(type = 'puller' and customerSource = true, lagduration, 0)) AS total_lag_duration_puller,
		SUM(IF(type = 'puller' and customerSource = true, costtime, 0)) AS total_cost_time_puller,
		ROUND(SUM(IF(type = 'puller' and customerSource = true, lagduration, 0)) * 100 / NULLIF(SUM(IF(type = 'puller' and customerSource = true, costtime, 0)), 0), 1) AS ratio_lag_puller,

		SUM(IF(type = 'internal-player' and customerSource = false, lagduration, 0)) AS total_lag_duration_internal_player,
		SUM(IF(type = 'internal-player' and customerSource = false, costtime, 0)) AS total_cost_time_internal_player,
		ROUND(SUM(IF(type = 'internal-player' and customerSource = false, lagduration, 0)) * 100 / NULLIF(SUM(IF(type = 'internal-player' and customerSource = false, costtime, 0)), 0), 1) AS ratio_lag_internal_player,

		SUM(IF(type = 'publisher', lagduration, 0)) AS total_lag_duration_publisher,
		SUM(IF(type = 'publisher', costtime, 0)) AS total_cost_time_publisher,
		ROUND(SUM(IF(type = 'publisher', lagduration, 0)) / NULLIF(SUM(IF(type = 'publisher', costtime, 0)), 0), 4) AS ratio_lag_publisher,

		COUNT(DISTINCT IF(type = 'puller' and customerSource != true, requestid, NULL)) AS requests_puller,
		COUNT(DISTINCT IF(type = 'puller' AND retryTimes > 0 and customerSource != true, requestid, NULL)) AS retry_requests_puller,
		ROUND(COUNT(DISTINCT IF(type = 'puller' AND retryTimes > 0 and customerSource != true, requestid, NULL)) * 100 / NULLIF(COUNT(DISTINCT IF(type = 'puller' and customerSource != true, requestid, NULL)), 0), 1) AS retry_ratio_puller,

		SUM(if(type = 'puller' and customerSource != true, retryTimes, 0)) as totalRetryTimes
	FROM dwd_flowd_miku_streamd_log
	WHERE 
	from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s' AND TIMESTAMP '%s'
	and AppName = '%s' 
	and day >= '%s' and day <= '%s'`, req.StartTime, req.EndTime, req.AppName, startDay, endDay)
	if req.StreamID != "" {
		if req.FuzzySearch {
			sql += fmt.Sprintf(" AND StreamName like '%%%s%%'\n", req.StreamID)
		} else {
			sql += fmt.Sprintf(" AND StreamName = '%s'\n", req.StreamID)
		}
	}
	sql += "GROUP BY date_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai')\n"
	sql += "ORDER BY ts_m"
	return sql
}

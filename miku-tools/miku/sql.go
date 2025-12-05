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
	pos := strings.Index(streamId, "_sxrxc")
	if pos > 0 {
		streamId = streamId[:pos]
	}
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
		startTime := strings.ReplaceAll(req.StartTime, "T", " ")
		sql += fmt.Sprintf(" AND cts >= to_unixtime(TIMESTAMP '%s+08:00')", startTime)
	}

	if req.EndTime != "" {
		endDay := s.convertToDay(req.EndTime)
		log.Printf("EndTime输入: %s, 转换后: %s", req.EndTime, endDay)
		sql += fmt.Sprintf(" AND day <= '%s'", endDay)
		endTime := strings.ReplaceAll(req.EndTime, "T", " ")
		sql += fmt.Sprintf(" AND cts <= to_unixtime(TIMESTAMP '%s+08:00')", endTime)
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
	sql += " LIMIT 50000"

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
	from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
	and AppName = '%s' 
	and day >= '%s' and day <= '%s'`, req.StartTime, req.EndTime, req.AppName, startDay, endDay)
	if req.StreamID != "" {
		if req.FuzzySearch {
			sql += fmt.Sprintf(" AND StreamName like '%%%s%%'\n", req.StreamID)
		} else {
			sql += fmt.Sprintf(" AND StreamName = '%s'\n", req.StreamID)
		}
	}
	sql += " GROUP BY date_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai')\n"
	sql += " ORDER BY ts_m"
	return sql
}

func (s *QOSServer) buildMikuFpsSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := s.convertToDay(req.StartTime)
	endDay := s.convertToDay(req.EndTime)

	sql := fmt.Sprintf(`
		WITH all_data AS (
			SELECT 
				from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai' as ts,
				StreamName,
				NodeID,
				-- 使用 reduce 函数计算数组平均值
				CASE 
				WHEN cardinality(Fps) = 0 THEN 0
				ELSE reduce(Fps, CAST(0 AS double), (s, x) -> s + x, s -> s) / cardinality(Fps)
				END AS avg_IncomingVideoFps,
				CASE 
				WHEN cardinality(AudioFps) = 0 THEN 0
				ELSE reduce(AudioFps, CAST(0 AS double), (s, x) -> s + x, s -> s) / cardinality(AudioFps)
				END AS avg_IncomingAudioFps,
				'publisher' AS source_type,
				1 AS priority
			FROM miku.dwd_flowd_miku_streamd_log
			WHERE             
				AppName = '%s'
				AND from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
				AND day >= '%s' AND day <= '%s'
				AND StreamName = '%s'
				AND Type = 'publisher'

			UNION ALL

			SELECT 
				from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai' as ts,
				StreamName,
				NodeID,
				CASE 
				WHEN cardinality(Fps) = 0 THEN 0
				ELSE reduce(Fps, CAST(0 AS double), (s, x) -> s + x, s -> s) / cardinality(Fps)
				END AS avg_IncomingVideoFps,
				CASE 
				WHEN cardinality(AudioFps) = 0 THEN 0
				ELSE reduce(AudioFps, CAST(0 AS double), (s, x) -> s + x, s -> s) / cardinality(AudioFps)
				END AS avg_IncomingAudioFps,
				'puller' AS source_type,
				2 AS priority
			FROM miku.dwd_flowd_miku_streamd_log
			WHERE            
				AppName = '%s'
				AND from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
				AND day >= '%s' AND day <= '%s'
				AND StreamName = '%s'
				AND Type = 'puller'
				AND CustomerSource = true
			),
				
			real_data AS (
			SELECT *
			FROM (
				SELECT *,
				row_number() OVER (PARTITION BY ts ORDER BY priority) AS rn
				FROM all_data
			)
			WHERE rn = 1
			)

			SELECT 
			r.ts,
			r.NodeID,
			r.StreamName,
			COALESCE(r.avg_IncomingVideoFps, 0) AS avg_IncomingVideoFps,
			COALESCE(r.avg_IncomingAudioFps, 0) AS avg_IncomingAudioFps,
			r.source_type
			FROM real_data r
			ORDER BY r.ts
		`, req.AppName, req.StartTime, req.EndTime, startDay, endDay, req.StreamID,
		req.AppName, req.StartTime, req.EndTime, startDay, endDay, req.StreamID)
	return sql
}

func (s *QOSServer) buildMikuStreamCntSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := s.convertToDay(req.StartTime)
	endDay := s.convertToDay(req.EndTime)

	sql := fmt.Sprintf(`
		select date_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai') as ts_m, count(DISTINCT streamname) as stream_cnt
		from dwd_flowd_miku_streamd_log 
		where from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
			and AppName = '%s'
			and day >= '%s' and day <= '%s'
		group by date_trunc('minute', from_unixtime(ts/1000000000) at time zone 'Asia/Shanghai')
		`, req.StartTime, req.EndTime, req.AppName, startDay, endDay)

	return sql
}

func (s *QOSServer) buildMikuUpstreamBandwidthSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := s.convertToDay(req.StartTime)
	endDay := s.convertToDay(req.EndTime)

	sql := fmt.Sprintf(`
		WITH all_data AS (
		SELECT 
			Ts AS ts,
			StreamName,
			NodeID,
			IF(CostTime = 0, 0, RecvBytes * 8 * 1000 / 1000 / 1000 / CostTime) AS bandwidth,
			'publisher' AS source_type,
			1 AS priority
		FROM miku.dwd_flowd_miku_streamd_log
		WHERE             
			AppName = '%s'
			AND from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
			AND day >= '%s' AND day <= '%s'
			AND StreamName = '%s'
			AND Type = 'publisher'
		
		UNION ALL
		
		SELECT 
			Ts AS ts,
			StreamName,
			NodeID,
			IF(CostTime = 0, 0, RecvBytes * 8 * 1000 / 1000 / 1000 / CostTime) AS bandwidth,
			'puller' AS source_type,
			2 AS priority
		FROM miku.dwd_flowd_miku_streamd_log
		WHERE            
			AppName = '%s'
			AND from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
			AND day >= '%s' AND day <= '%s'
			AND StreamName = '%s'
			AND Type = 'puller'
			AND CustomerSource = true
		),
			
		real_data AS (
		SELECT *
		FROM (
			SELECT *,
			row_number() OVER (PARTITION BY ts ORDER BY priority) AS rn
			FROM all_data
		)
		WHERE rn = 1
		)
		
		SELECT 
		r.ts,
		r.NodeID,
		r.StreamName,
		r.bandwidth,
		r.source_type
		FROM real_data r
		ORDER BY r.ts
		`, req.AppName, req.StartTime, req.EndTime, startDay, endDay, req.StreamID,
		req.AppName, req.StartTime, req.EndTime, startDay, endDay, req.StreamID)

	return sql
}

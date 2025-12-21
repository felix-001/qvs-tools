package qos

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// buildSQLQuery 构建SQL查询语句
func BuildSQLQuery(req QOSRequest, raw bool) string {

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
		startDay := convertToDay(req.StartTime)
		log.Printf("StartTime输入: %s, 转换后: %s", req.StartTime, startDay)
		sql += fmt.Sprintf(" AND day >= '%s'", startDay)
		startTime := strings.ReplaceAll(req.StartTime, "T", " ")
		sql += fmt.Sprintf(" AND cts >= to_unixtime(TIMESTAMP '%s+08:00')", startTime)
	}

	if req.EndTime != "" {
		endDay := convertToDay(req.EndTime)
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

func BuildMikuSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

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
		COUNT(DISTINCT IF(type = 'puller' AND retryTimes > 0 and customerSource != true and url not like '%%ffmpegplayer%%', requestid, NULL)) AS retry_requests_puller,
		ROUND(COUNT(DISTINCT IF(type = 'puller' AND retryTimes > 0 and url not like '%%ffmpegplayer%%' and customerSource != true, requestid, NULL)) * 100 / NULLIF(COUNT(DISTINCT IF(type = 'puller' and customerSource != true, requestid, NULL)), 0), 1) AS retry_ratio_puller,

		SUM(if(type = 'puller' and customerSource != true and url not like '%%ffmpegplayer%%', retryTimes, 0)) as totalRetryTimes,

		SUM(if(type = 'puller' and customerSource = true and url not like '%%ffmpegplayer%%', retryTimes, 0)) as upstreamRetryTimes,
		ROUND(COUNT(DISTINCT IF(type = 'puller' AND retryTimes > 0 and url not like '%%ffmpegplayer%%' and customerSource = true, requestid, NULL)) * 100 / NULLIF(COUNT(DISTINCT IF(type = 'puller' and customerSource = true, requestid, NULL)), 0), 1) AS retry_ratio_upstream,

		COUNT(CASE WHEN SendFirstPktTime < 1000 THEN 1 END) as loadCnt,
		COUNT(*) as total,
		COUNT(CASE WHEN SendFirstPktTime < 1000 THEN 1 END) * 100.0 / COUNT(*) as loadRatio
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

func buildMikuFpsSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

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

func BuildMikuStreamCntSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

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

func BuildMikuUpstreamBandwidthSQLQuery(req QOSRequest) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

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

func buildCommonSQLQuery(req QOSRequest, choose, table, where, group, order string) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

	sql := fmt.Sprintf(`
		SELECT 
		    %s
		FROM %s
		WHERE 1=1 
			%s
			AND from_unixtime(cts) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
			AND dim_heart_type != '0'
			AND (client_type = 'sdk_video_bad_quality_ratio' OR client_type = 'web_video_bad_quality_ratio')
			AND day >= '%s' AND day <= '%s'
		`, choose, table, where, req.StartTime, req.EndTime, startDay, endDay)
	if group != "" {
		sql += fmt.Sprintf(" GROUP BY %s", group)
	}
	if order != "" {
		sql += fmt.Sprintf(" ORDER BY %s", order)
	}
	return sql

}

func buildMikuCommonSQLQuery(req QOSRequest, choose, where, group, order string) string {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

	if req.StreamID != "" {
		if req.FuzzySearch {
			where += fmt.Sprintf(`AND StreamName like '%%%s%%'`, req.StreamID)
		} else {
			where += fmt.Sprintf(`AND StreamName = '%s'`, req.StreamID)
		}
	}

	sql := fmt.Sprintf(`
		SELECT 
		    %s
		FROM %s
		WHERE 1=1 
			%s
			AND from_unixtime(ts/1000000000) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
			AND day >= '%s' AND day <= '%s'
		`, choose, "miku.dwd_flowd_miku_streamd_log", where, req.StartTime, req.EndTime, startDay, endDay)
	if group != "" {
		sql += fmt.Sprintf(" GROUP BY %s", group)
	}
	if order != "" {
		sql += fmt.Sprintf(" ORDER BY %s", order)
	}
	return sql
}

// protocol: "hls" "p2p" "flv"
func buildHyCommonSQLQuery(req QOSRequest, choose, where, group, order, protocol string) string {
	if req.Domain != "" {
		if req.FuzzySearch {
			where += fmt.Sprintf(`
			AND dim_stream_url like '%%%s%%'
			AND (client_type = 'sdk_video_bad_quality_ratio' or client_type = 'web_video_bad_quality_ratio')
		`, req.Domain)
		} else {
			where += fmt.Sprintf(`
			AND dim_stream_url = '%s'
			AND (client_type = 'sdk_video_bad_quality_ratio' or client_type = 'web_video_bad_quality_ratio')
	`, req.Domain)
		}
	}
	if req.StreamID != "" {
		if req.FuzzySearch {
			where += fmt.Sprintf(`AND dim_stream_url like '%%%s%%'`, req.StreamID)
		} else {
			where += fmt.Sprintf(`AND dim_stream_url = '%s'`, req.StreamID)
		}
	}
	switch protocol {
	case "hls":
	case "p2p":
		where += ` AND dim_p2p = '1'`
	case "flv":
		/*/
		where += fmt.Sprintf(`
			AND dim_stream_url like '%%.flv%%'
		`)
		*/
		where += ` AND dim_p2p = '0'`
	}
	return buildCommonSQLQuery(req, choose, "miku.huyabiz_quality_report_log", where, group, order)
}

func buidUpstreamDistributeSQLQuery(req QOSRequest) string {
	choose := `DISTINCT REGEXP_EXTRACT(RemoteAddr, '^(?:\[)?([0-9a-fA-F:.]+)(?:\])?:\d+$', 1) as RemoteAddr`
	where := fmt.Sprintf(`
        	AND AppName = '%s'
  		AND CustomerSource = true
		AND HTTPResponseCode != 302
	`, req.AppName)

	return buildMikuCommonSQLQuery(req, choose, where, "", "")
}

func BuildHyCdnLagSQLQuery(req QOSRequest) string {
	choose := `
		dim_cdnip,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt,
		COUNT(*) as total	
		`
	return buildHyCommonSQLQuery(req, choose, "", "dim_cdnip", "lagCnt DESC", req.Protocol)
}

func BuildHyClientLagSQLQuery(req QOSRequest) string {
	choose := `
		dim__ip,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt,
		COUNT(*) as total	
		`
	return buildHyCommonSQLQuery(req, choose, "", "dim__ip", "lagCnt DESC", req.Protocol)
}

func BuildClientIpsOnCdnIpsSQLQuery(req QOSRequest) string {
	/*
		choose := `
			dim_cdnip, dim__ip, dim_stream_url,
			COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt
			`
	*/
	choose := `
		dim_cdnip, dim__ip,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt
		`
	//return buildHyCommonSQLQuery(req, choose, "", "dim_cdnip, dim__ip, dim_stream_url", "", req.Protocol)
	return buildHyCommonSQLQuery(req, choose, "", "dim_cdnip, dim__ip", "", req.Protocol)
}

func moreThan1day(start, end string) bool {
	timeLayout := "2006-01-02T15:04:05"
	startTime, err1 := time.Parse(timeLayout, start)
	endTime, err2 := time.Parse(timeLayout, end)

	if err1 != nil || err2 != nil {
		log.Println("时间格式错误:", err1, err2)
		return false
	} else {
		// Check if time difference is at least 1 day (86400 seconds)
		if endTime.Sub(startTime).Seconds() >= 86400 {
			return true
		}
	}
	return false
}

func BuildHyLagRateSQLQuery(req QOSRequest) string {
	ts := "date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai') as ts_m,"
	order := "ts_m"
	group := "date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')"
	if moreThan1day(req.StartTime, req.EndTime) {
		ts = "from_unixtime(floor(cts / 600) * 600) at time zone 'Asia/Shanghai' as ts_m,"
		group = "from_unixtime(floor(cts / 600) * 600) at time zone 'Asia/Shanghai'"
	}

	choose := ts + `
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt,
		COUNT(*) as total,

		COUNT(DISTINCT IF(field_video_bad_quality = 100 , dim__ip, NULL)) AS lag_usr_cnt,	
		COUNT(DISTINCT dim__ip) AS total_usr_cnt,	
		COUNT(DISTINCT IF(field_video_bad_quality = 100 , dim__ip, NULL)) * 100.0 / COUNT(DISTINCT dim__ip) AS lag_usr_rate,

		COUNT(DISTINCT IF(field_video_bad_quality = 100 , dim_cdnip, NULL)) AS lag_node_cnt,	
		COUNT(DISTINCT dim_cdnip) AS total_node_cnt,	
		COUNT(DISTINCT IF(field_video_bad_quality = 100 , dim_cdnip, NULL)) * 100.0 / COUNT(DISTINCT dim_cdnip) AS lag_node_rate,

		COUNT(CASE WHEN field_video_bad_quality = 100 and (dim_stream_url like '%ratio%' or dim_stream_url like '%codec%') and dim_stream_url not like '%src%' THEN 1 END) as transcodeLagCnt,
		COUNT(CASE WHEN (dim_stream_url like '%ratio%' or dim_stream_url like '%codec%') and dim_stream_url not like '%src%' THEN 1 END) as totalTranscodeCnt,
		COALESCE(
			COUNT(CASE WHEN field_video_bad_quality = 100 AND (dim_stream_url LIKE '%ratio%' OR dim_stream_url LIKE '%codec%') AND dim_stream_url NOT LIKE '%src%' THEN 1 END) * 100.0 /
			NULLIF(COUNT(CASE WHEN (dim_stream_url LIKE '%ratio%' OR dim_stream_url LIKE '%codec%') AND dim_stream_url NOT LIKE '%src%' THEN 1 END), 0),
			0
			) as trancodeLagRate,
		
		COUNT(CASE WHEN field_video_bad_quality = 100 and (dim_stream_url not like '%ratio%' and dim_stream_url not like '%codec%') THEN 1 END) as notTranscodeLagCnt,
		COUNT(CASE WHEN (dim_stream_url not like '%ratio%' or dim_stream_url not like '%codec%') THEN 1 END) as totalNotTranscodeCnt,
		COALESCE(
			COUNT(CASE WHEN field_video_bad_quality = 100 AND (dim_stream_url NOT LIKE '%ratio%' AND dim_stream_url NOT LIKE '%codec%') THEN 1 END) * 100.0 /
			NULLIF(COUNT(CASE WHEN (dim_stream_url NOT LIKE '%ratio%' AND dim_stream_url NOT LIKE '%codec%') THEN 1 END), 0),
			0
		) as notTranscodeLagRate,
		
		COALESCE(
			COUNT(CASE WHEN field_video_bad_quality = 100 AND (dim_stream_url LIKE '%ratio%' OR dim_stream_url LIKE '%codec%') AND dim_stream_url NOT LIKE '%src%' THEN 1 END) * 100.0 /
			NULLIF(COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END), 0),
			0
			) as trancodeLagPercent
		`
	return buildHyCommonSQLQuery(req, choose, "", group, order, req.Protocol)
}

func BuildHyCdnIpLagSQLQuery(req QOSRequest) string {
	choose := `
		date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai') as ts_m,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent,
		COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt,
		COUNT(*) as total
	`
	where := fmt.Sprintf(`
		AND dim_cdnip = '%s'
	`, req.CdnIp)
	return buildHyCommonSQLQuery(req, choose, where, "date_trunc('minute', from_unixtime(cts) at time zone 'Asia/Shanghai')", "ts_m", "flv")
}

func BuildHyLagRateByStreamsSQLQuery(req QOSRequest) string {
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

	p2p := ""
	switch req.Protocol {
	case "hls":
	case "p2p":
		p2p = ` AND dim_p2p = '1'`
	case "flv":
		/*/
		where += fmt.Sprintf(`
			AND dim_stream_url like '%%.flv%%'
		`)
		*/
		p2p = ` AND dim_p2p = '0'`
	}

	sql := fmt.Sprintf(`
WITH extracted_parts AS (
  SELECT
    -- 提取文件名部分
    regexp_extract(dim_stream_url, '[src|huyap2p|huyalive|huyacdn|huyacdntest]/([^?]+)\.[flv|slice]', 1) AS streamName,
    -- 提取ratio参数
    regexp_extract(dim_stream_url, '[?&]ratio=([^&]+)', 1) AS ratio_value,
    -- 提取codec参数
    regexp_extract(dim_stream_url, '[?&]codec=([^&]+)', 1) AS codec_value,
    *
  FROM huyabiz_quality_report_log
)

SELECT
  CONCAT(
    streamName,
    -- 如果有ratio值则添加
    CASE WHEN ratio_value IS NOT NULL THEN CONCAT('_', ratio_value) ELSE '' END,
    -- 如果有codec值则添加
    CASE WHEN codec_value IS NOT NULL THEN CONCAT('_', codec_value) ELSE '' END
  ) AS stream_id,
  COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent,
  COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) as lagCnt,
  COUNT(*) as total
FROM extracted_parts
WHERE 1=1  
	AND day >= '%s' AND day <= '%s' 
	AND cts >= to_unixtime(TIMESTAMP '%s+08:00') AND cts <= to_unixtime(TIMESTAMP '%s+08:00') 
	%s
	AND dim_stream_url like '%%%s%%'
	AND dim_heart_type != '0'
	AND (client_type = 'sdk_video_bad_quality_ratio' or client_type = 'web_video_bad_quality_ratio') 
group by  CONCAT(
    streamName,
    -- 如果有ratio值则添加
    CASE WHEN ratio_value IS NOT NULL THEN CONCAT('_', ratio_value) ELSE '' END,
    -- 如果有codec值则添加
    CASE WHEN codec_value IS NOT NULL THEN CONCAT('_', codec_value) ELSE '' END
  )
ORDER by lagCnt DESC
	`, startDay, endDay, req.StartTime, req.EndTime, p2p, req.StreamID)

	return sql
}

func BuildHyUsrDistributeSQLQuery(req QOSRequest) string {
	choose := `
		DISTINCT dim__ip
		`
	return buildHyCommonSQLQuery(req, choose, "", "", "", "flv")
}

func buildHyCommonLoadSQLQuery(req QOSRequest, choose, where, group, order string) string {
	// Replace "T" with space in starttime and endtime
	req.StartTime = strings.ReplaceAll(req.StartTime, "T", " ")
	req.EndTime = strings.ReplaceAll(req.EndTime, "T", " ")
	startDay := convertToDay(req.StartTime)
	endDay := convertToDay(req.EndTime)

	if req.StreamID != "" {
		if req.FuzzySearch {
			where += fmt.Sprintf(" AND dim_stream_url LIKE '%%%s%%'", req.StreamID)
		} else {
			where += fmt.Sprintf(" AND dim_stream_url = '%s'", req.StreamID)
		}
	}

	switch req.Protocol {
	case "hls":
	case "p2p":
		where += ` AND dim_p2p = '1'`
	case "flv":
		where += ` AND dim_p2p = '0'`
	}

	sql := fmt.Sprintf(`
		SELECT 
		    %s
		FROM %s
		WHERE 1=1 
			%s
			AND from_unixtime(cts) BETWEEN TIMESTAMP '%s+08:00' AND TIMESTAMP '%s+08:00'
			AND (client_type = 'sdk_video_load' OR client_type = 'web_video_load')
			AND day >= '%s' AND day <= '%s'
		`, choose, "huyabiz_quality_report_log", where, req.StartTime, req.EndTime, startDay, endDay)
	if group != "" {
		sql += fmt.Sprintf(" GROUP BY %s", group)
	}
	if order != "" {
		sql += fmt.Sprintf(" ORDER BY %s", order)
	}
	return sql

}

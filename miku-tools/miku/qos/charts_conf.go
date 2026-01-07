package qos

var Charts_conf_json = `
[
	{
		"name": "chart_hyStreamsLagRateTrend",
		"title": "虎牙多个流卡顿率趋势图",
		"seriesTitle": "卡顿率",
		"type": "line",
		"table": "huyabiz_quality_report_log",
		"sql": {
			"with": "extracted_parts AS (
					SELECT
						regexp_extract(dim_stream_url, '[src|huyap2p|huyalive|huyacdn|huyacdntest]/([^?]+)\.[flv|slice]', 1) AS streamName,
						regexp_extract(dim_stream_url, '[?&]ratio=([^&]+)', 1) AS ratio_value,
						regexp_extract(dim_stream_url, '[?&]codec=([^&]+)', 1) AS codec_value,
						*
					FROM huyabiz_quality_report_log)",
			"select": "CASE
					-- 如果streamName已经包含cxdexxtpl，则直接使用streamName
					WHEN streamName LIKE '%%cxdexxtpl%%' THEN streamName
					-- 否则，构造新的流ID
					ELSE
					-- 基础部分：streamName + 如果包含/src/则加_cxdexxtpl_huyaxsrcx
					CONCAT(
						streamName,
						CASE WHEN dim_stream_url LIKE '%%/src/%%' THEN '_sxrxc' ELSE '' END
					) ||
					-- 参数部分：只有当至少有一个参数不为空时才添加

						CASE
							WHEN ratio_value IS NOT NULL OR codec_value IS NOT NULL THEN
							CONCAT(
							'_cxdexxtpl_huyaxsrcx_',
							COALESCE(codec_value, 'null'),
							'_',
							COALESCE(ratio_value, 'null')
							)
							ELSE ''  -- 两个参数都为空，什么都不加
						END
					END AS stream_id,
					COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent",
			"where": "dim_stream_url LIKE '%/src/%' OR dim_stream_url LIKE '%sxrxc%'",
			"group_by": "CASE
					-- 如果streamName已经包含cxdexxtpl，则直接使用streamName
					WHEN streamName LIKE '%%cxdexxtpl%%' THEN streamName
					-- 否则，构造新的流ID
					ELSE
					-- 基础部分：streamName + 如果包含/src/则加_cxdexxtpl_huyaxsrcx
					CONCAT(
						streamName,
						CASE WHEN dim_stream_url LIKE '%%/src/%%' THEN '_sxrxc' ELSE '' END
					) ||
					-- 参数部分：只有当至少有一个参数不为空时才添加

						CASE
							WHEN ratio_value IS NOT NULL OR codec_value IS NOT NULL THEN
							CONCAT(
							'_cxdexxtpl_huyaxsrcx_',
							COALESCE(codec_value, 'null'),
							'_',
							COALESCE(ratio_value, 'null')
							)
							ELSE ''  -- 两个参数都为空，什么都不加
						END
					END",
			"order_by": "",
			"fields": ["percent"],
			"dimension": "stream_id"
		},
	}
]
`

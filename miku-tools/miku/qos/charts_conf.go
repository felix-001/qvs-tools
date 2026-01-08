package qos

var Charts_conf_json = `
[
	{
		"name": "chart_hyStreamsLagRateTrend",
		"title": "虎牙多个流卡顿率趋势图",
		"seriesTitle": "卡顿率",
		"type": "line",
		"table": "hy",
		"sql": {
			"with": "extracted_parts AS (SELECT regexp_extract(dim_stream_url, '[src|huyap2p|huyalive|huyacdn|huyacdntest]/([^?]+)\\.[flv|slice]', 1) AS streamName, regexp_extract(dim_stream_url, '[?&]ratio=([^&]+)', 1) AS ratio_value, regexp_extract(dim_stream_url, '[?&]codec=([^&]+)', 1) AS codec_value, * FROM huyabiz_quality_report_log)",
			"select": "CASE WHEN streamName LIKE '%cxdexxtpl%' THEN streamName ELSE CONCAT(streamName, CASE WHEN dim_stream_url LIKE '%/src/%' THEN '_sxrxc' ELSE '' END) || CASE WHEN ratio_value IS NOT NULL OR codec_value IS NOT NULL THEN CONCAT('_cxdexxtpl_huyaxsrcx_', COALESCE(codec_value, 'null'), '_', COALESCE(ratio_value, 'null')) ELSE '' END END AS stream_id, COUNT(CASE WHEN field_video_bad_quality = 100 THEN 1 END) * 100.0 / COUNT(*) as percent",
			"from": "extracted_parts",
			"where": "(dim_stream_url LIKE '%/src/%' OR dim_stream_url LIKE '%sxrxc%')",
			"group_by": "CASE WHEN streamName LIKE '%cxdexxtpl%' THEN streamName ELSE CONCAT(streamName, CASE WHEN dim_stream_url LIKE '%/src/%' THEN '_sxrxc' ELSE '' END) || CASE WHEN ratio_value IS NOT NULL OR codec_value IS NOT NULL THEN CONCAT('_cxdexxtpl_huyaxsrcx_', COALESCE(codec_value, 'null'), '_', COALESCE(ratio_value, 'null')) ELSE '' END END",
			"group_by_minute": true,
			"order_by": "ts_m",
			"field": "percent",
			"dimension": "stream_id"
		}
	}
]
`

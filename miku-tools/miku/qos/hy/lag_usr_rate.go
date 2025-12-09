package hy

import (
	"mikutool/miku/qos"
)

// 卡顿用户占总用户数的百分比

func init() {
	qos.RegisterChartGenerator("lag_usr_rate", &LagUsrRate{})
}

type LagUsrRate struct {
}

func (l *LagUsrRate) Generate(req qos.QOSRequest) string {
	// COUNT(DISTINCT IF(type = 'puller' and customerSource != true, requestid, NULL)) AS requests_puller,
	//	COUNT(DISTINCT IF(type = 'puller' AND retryTimes > 0 and customerSource != true and url not like '%%ffmpegplayer%%', requestid, NULL)) AS retry_requests_puller,
	//	ROUND(COUNT(DISTINCT IF(type = 'puller' AND retryTimes > 0 and url not like '%%ffmpegplayer%%' and customerSource != true, requestid, NULL)) * 100 / NULLIF(COUNT(DISTINCT IF(type = 'puller' and customerSource != true, requestid, NULL)), 0), 1) AS retry_ratio_puller,
	// 跟hy lag rate查询合并到一起，减少查询次数
	return ""
}

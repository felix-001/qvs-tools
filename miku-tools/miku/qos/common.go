package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"
	"strconv"
	"time"
)

func convertToDay(timeStr string) string {
	// 尝试解析不同格式的时间字符串
	layouts := []string{
		"2006-01-02T15:04:05", // datetime-local格式
		"2006-01-02 15:04:05", // 标准日期时间格式
		"2006-01-02",          // 仅日期格式
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			result := t.Format("20060102") // 20251203格式
			//log.Printf("时间转换成功: 输入=%s, 布局=%s, 结果=%s", timeStr, layout, result)
			return result
		}
	}

	// 如果解析失败，直接返回原始字符串
	log.Printf("时间转换失败，返回原始字符串: %s", timeStr)
	return timeStr
}

// getString 安全获取字符串指针值
func getString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

// getInt64 安全获取int64指针值
func getInt64(i *int64) string {
	if i == nil {
		return ""
	}
	return strconv.FormatInt(*i, 10)
}

type HyLagRateCache struct {
	HyLagRateReports []util.HyLagReport
	Req              QOSRequest
}

var hyLagRateCache HyLagRateCache

func GetHyLagRateReports(req QOSRequest) ([]util.HyLagReport, error) {
	if req.StartTime == hyLagRateCache.Req.StartTime && req.EndTime == hyLagRateCache.Req.EndTime &&
		len(hyLagRateCache.HyLagRateReports) > 0 {
		return hyLagRateCache.HyLagRateReports, nil
	}
	sql := BuildHyLagRateSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &hyLagRateCache.HyLagRateReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	hyLagRateCache.Req = req
	//bytes, _ := json.Marshal(hyLagRateCache)
	//fmt.Printf("HyLagRateReports: %+v\n", string(bytes))
	return hyLagRateCache.HyLagRateReports, nil
}

type HyDistinctCdnClientIpsCache struct {
	HyClientIpsOnCdnIpReports []util.HyClientIpsOnCdnIpReport
	Req                       QOSRequest
}

var hyDistinctCdnClientIpsCache HyDistinctCdnClientIpsCache

func GetHyDistinctCdnClientIps(req QOSRequest) ([]util.HyClientIpsOnCdnIpReport, error) {
	if req.StartTime == hyDistinctCdnClientIpsCache.Req.StartTime &&
		req.EndTime == hyDistinctCdnClientIpsCache.Req.EndTime &&
		len(hyDistinctCdnClientIpsCache.HyClientIpsOnCdnIpReports) > 0 {
		return hyDistinctCdnClientIpsCache.HyClientIpsOnCdnIpReports, nil
	}
	sql := BuildClientIpsOnCdnIpsSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &hyDistinctCdnClientIpsCache.HyClientIpsOnCdnIpReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return nil, err
	}
	return hyDistinctCdnClientIpsCache.HyClientIpsOnCdnIpReports, nil
}

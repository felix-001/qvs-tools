package qos

import (
	"fmt"
	"log"
	"mikutool/public/util"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qbox/pili/common/ipdb.v1"
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

var hyStreamdLagCache struct {
	StreamdLagReports []util.StreamdLagReport
	Req               QOSRequest
}

func GetMikuStreamdReportLagDatas(req QOSRequest) ([]util.StreamdLagReport, error) {
	if req.StartTime == hyStreamdLagCache.Req.StartTime && req.EndTime == hyStreamdLagCache.Req.EndTime &&
		len(hyStreamdLagCache.StreamdLagReports) > 0 {
		return hyStreamdLagCache.StreamdLagReports, nil
	}
	sql := BuildMikuSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &hyStreamdLagCache.StreamdLagReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	hyStreamdLagCache.Req = req
	return hyStreamdLagCache.StreamdLagReports, nil
}

func AggUpstreamDistributeReports(reports []util.UpstreamDistributeReport, ipparser *ipdb.City) (map[string]int, map[string]int) {
	areaCntMap := make(map[string]int)
	provCntMap := make(map[string]int)
	for _, report := range reports {
		if report.RemoteAddr == nil {
			continue
		}
		_, _, area, prov := util.GetLocate(*report.RemoteAddr, ipparser)
		areaCntMap[area]++
		provCntMap[prov]++
	}
	return areaCntMap, provCntMap
}

var hyUpstreamDistributeCache struct {
	UpstreamDistributeReports []util.UpstreamDistributeReport
	Req                       QOSRequest
}

func GetUpstreamDistributeReport(req QOSRequest) ([]util.UpstreamDistributeReport, error) {
	if req.StartTime == hyUpstreamDistributeCache.Req.StartTime && req.EndTime == hyUpstreamDistributeCache.Req.EndTime &&
		len(hyUpstreamDistributeCache.UpstreamDistributeReports) > 0 {
		return hyUpstreamDistributeCache.UpstreamDistributeReports, nil
	}
	sql := buidUpstreamDistributeSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &hyUpstreamDistributeCache.UpstreamDistributeReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	hyUpstreamDistributeCache.Req = req
	return hyUpstreamDistributeCache.UpstreamDistributeReports, nil
}

func SortData(data map[string]int) []PieDataItem {
	// 排序数据（按值降序）

	var sortedData []PieDataItem
	for name, value := range data {
		sortedData = append(sortedData, PieDataItem{Name: name, Value: value})
	}

	sort.Slice(sortedData, func(i, j int) bool {
		return sortedData[i].Value > sortedData[j].Value
	})

	// 限制最多显示10个
	if len(sortedData) > 10 {
		sortedData = sortedData[:10]
	}
	return sortedData
}

func GetAggData(req QOSRequest, reports []util.HyClientIpsOnCdnIpReport) AggData {
	clientIpMap := make(map[string]bool)
	lagIpMap := make(map[string]bool)
	for _, report := range reports {
		if report.DimIp == nil {
			continue
		}
		clientIpMap[*report.DimIp] = true
		if report.LagCnt != nil && *report.LagCnt > 0 {
			lagIpMap[*report.DimIp] = true
		}
	}
	countryMap := make(map[string]int)
	areaMap := make(map[string]int)
	provinceMap := make(map[string]int)
	areaLagMap := make(map[string]int)
	provLagMap := make(map[string]int)
	for clientIp := range clientIpMap {
		country, _, area, province := util.GetLocate(clientIp, req.IpParser)
		countryMap[country]++
		areaMap[area]++
		provinceMap[province]++
		if _, ok := lagIpMap[clientIp]; ok {
			areaLagMap[area]++
		}
		if _, ok := lagIpMap[clientIp]; ok {
			provLagMap[province]++
		}
	}
	aggData := AggData{
		CountryCntMap: countryMap,
		AreaCntMap:    areaMap,
		ProvCntMap:    provinceMap,
		AreaLagCntMap: areaLagMap,
		ProvLagCntMap: provLagMap,
	}
	return aggData
}

type Cb func(ts string, value any)

func GenerateLineChartData(title, seriesTitle string, traverseFn func(cb Cb)) LineChartData {
	data := LineChartData{
		Title:       title,       //"虎牙每分钟卡顿率趋势图",
		SeriesTitle: seriesTitle, //"卡顿率",
		Color:       "#1890ff",
		XType:       "category",
		XAxis:       []string{},
		YAxis:       []string{},
	}
	cb := func(ts string, value any) {
		ts = strings.ReplaceAll(ts, "+08:00", "")
		// Remove year from timestamp (format: 2025-12-16T12:49:00)
		if len(ts) >= 10 {
			ts = ts[5:] // Keep everything after the year
		}
		data.XAxis = append(data.XAxis, ts)
		if num, ok := value.(float64); ok {
			data.YAxis = append(data.YAxis, fmt.Sprintf("%.2f", num))
		} else if num, ok := value.(int); ok {
			data.YAxis = append(data.YAxis, fmt.Sprintf("%d", num))
		}
	}
	traverseFn(cb)
	return data
}

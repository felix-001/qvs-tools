package qos

import (
	"log"
	"mikutool/public/util"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/qbox/pili/common/ipdb.v1"
)

// aggregateByMinute 按分钟聚合数据（模拟第二个SQL查询的效果）
func aggregateByMinute(reports []util.QualityReport) []MinuteAggregatedData {
	// 按分钟分组聚合
	minuteMap := make(map[string]*MinuteAggregatedData)

	for _, report := range reports {
		// 获取时间戳并按分钟截断
		var timestamp time.Time
		if report.Cts != nil {
			timestamp = time.Unix(*report.Cts, 0).In(time.FixedZone("Asia/Shanghai", 8*60*60))
		} else if report.LogTime != nil {
			timestamp = time.Unix(*report.LogTime, 0).In(time.FixedZone("Asia/Shanghai", 8*60*60))
		} else {
			log.Println("get ts err")
			continue
		}

		// 按分钟截断时间
		truncatedTime := time.Date(
			timestamp.Year(), timestamp.Month(), timestamp.Day(),
			timestamp.Hour(), timestamp.Minute(), 0, 0,
			time.FixedZone("Asia/Shanghai", 8*60*60),
		).Format("2006-01-02 15:04:05")

		// 初始化分钟数据
		if _, exists := minuteMap[truncatedTime]; !exists {
			minuteMap[truncatedTime] = &MinuteAggregatedData{
				Timestamp:  truncatedTime,
				Percent:    0.0,
				LagCount:   0,
				TotalCount: 0,
			}
		}

		// 统计数据
		minuteData := minuteMap[truncatedTime]
		minuteData.TotalCount++
		minuteData.Reports = append(minuteData.Reports, report)

		// 检查是否为不良质量
		if report.FieldVideoBadQuality != nil && *report.FieldVideoBadQuality == 100 {
			minuteData.LagCount++
		}
	}

	// 计算百分比并转换为切片
	var result []MinuteAggregatedData
	var lagUserMap = make(map[string]bool)
	var totalUserMap = make(map[string]bool)
	var LagCdnMap = make(map[string]bool)
	var TotalCdnMap = make(map[string]bool)
	for _, data := range minuteMap {
		if data.TotalCount > 0 {
			data.Percent = float64(data.LagCount) / float64(data.TotalCount) * 100
		}
		for _, report := range data.Reports {
			if report.DimIp != nil && *report.DimIp != "" {
				if report.FieldVideoBadQuality != nil && *report.FieldVideoBadQuality == 100 {
					lagUserMap[*report.DimIp] = true
				}
				totalUserMap[*report.DimIp] = true
			}
			if report.DimCdnip != nil && *report.DimCdnip != "" {
				if report.FieldVideoBadQuality != nil && *report.FieldVideoBadQuality == 100 {
					LagCdnMap[*report.DimCdnip] = true
				}
				TotalCdnMap[*report.DimCdnip] = true
			}
		}
		data.LagUserCnt = len(lagUserMap)
		data.TotalUserCnt = len(totalUserMap)
		data.LagCdnCnt = len(LagCdnMap)
		data.TotalCdnCnt = len(TotalCdnMap)
		result = append(result, *data)
	}

	// 按时间排序
	sort.Slice(result, func(i, j int) bool {
		timeI, _ := time.Parse("2006-01-02 15:04:05", result[i].Timestamp)
		timeJ, _ := time.Parse("2006-01-02 15:04:05", result[j].Timestamp)
		return timeI.Before(timeJ)
	})

	return result
}

func aggregateReport(reports []util.QualityReport, ipparser *ipdb.City) ([]AggregatedData, []AggregatedData, []CdnAggregateData, AggData) {
	// CDN IP聚合
	cdnLagCntMap := make(map[string]int)   // cdnip -> 延迟数
	cdnTotalCntMap := make(map[string]int) // cdnip -> 总数
	cdnLagClientMap := make(map[string]map[string]bool)
	cdnAllClientMap := make(map[string]map[string]bool)

	// Client IP聚合
	clientLagCntMap := make(map[string]int)   // clientIp -> 延迟数
	clientTotalCntMap := make(map[string]int) // clientIp -> 总数
	clientCdnIpsMap := make(map[string]map[string]bool)

	areaLagCntMap := make(map[string]int) // area -> 延迟数
	provLagCntMap := make(map[string]int) // prov -> 延迟数

	for _, report := range reports {
		// 获取CDN IP
		var cdnip string
		if report.DimCdnip != nil && *report.DimCdnip != "" {
			cdnip = *report.DimCdnip
		}

		if (cdnip == "" || cdnip == "qn.flv.huya.com" || cdnip == "http://qn.flv.huya.com") &&
			report.DimStreamUrl != nil && *report.DimStreamUrl != "" {
			// 从URL中提取域名作为CDN IP
			if u, err := url.Parse(*report.DimStreamUrl); err == nil {
				host := u.Host
				if h, _, err := net.SplitHostPort(host); err == nil {
					cdnip = h
				} else {
					cdnip = host
				}
			}
		}

		// 获取Client IP
		var clientIp string
		if report.DimIp != nil {
			clientIp = *report.DimIp
		}

		// 获取不良质量状态
		var isBadQuality bool
		if report.FieldVideoBadQuality != nil && *report.FieldVideoBadQuality == 100 {
			isBadQuality = true
		}

		// 处理CDN IP聚合
		if cdnip != "" {
			cdnTotalCntMap[cdnip]++
			if isBadQuality {
				cdnLagCntMap[cdnip]++
			}
		}

		// 处理Client IP聚合
		if clientIp != "" {
			clientTotalCntMap[clientIp]++
			if isBadQuality {
				clientLagCntMap[clientIp]++
				_, _, area, prov := util.GetLocate(clientIp, ipparser)
				areaLagCntMap[area]++
				provLagCntMap[prov]++

			}
			if _, exists := clientCdnIpsMap[clientIp]; !exists {
				clientCdnIpsMap[clientIp] = make(map[string]bool)
			}
			clientCdnIpsMap[clientIp][cdnip] = true
		}

		if isBadQuality {
			if _, exists := cdnLagClientMap[cdnip]; !exists {
				cdnLagClientMap[cdnip] = make(map[string]bool)
			}
			cdnLagClientMap[cdnip][clientIp] = true
		}
		if _, exists := cdnAllClientMap[cdnip]; !exists {
			cdnAllClientMap[cdnip] = make(map[string]bool)
		}
		cdnAllClientMap[cdnip][clientIp] = true
	}

	// 生成CDN聚合数据
	var cdnAggregated []AggregatedData
	totalLagCdnCnt := 0
	for cdnip, total := range cdnTotalCntMap {
		if cdnip == "qn.flv.huya.com" {
			continue
		}
		clientIps := make([]string, 0)
		for clientIp := range cdnAllClientMap[cdnip] {
			clientIps = append(clientIps, clientIp)
		}
		_, isp, _, prov := util.GetLocate(cdnip, ipparser)
		cdnAggregated = append(cdnAggregated, AggregatedData{
			IP:         cdnip,
			TotalCount: total,
			LagCount:   cdnLagCntMap[cdnip],
			Prov:       prov,
			Isp:        isp,
			RemoteIps:  clientIps,
		})
		totalLagCdnCnt += cdnLagCntMap[cdnip]
	}
	for i := range cdnAggregated {
		cdnAggregated[i].LagRate = float64(cdnAggregated[i].LagCount*100) / float64(totalLagCdnCnt)
	}

	// 按延迟次数降序排序
	sort.Slice(cdnAggregated, func(i, j int) bool {
		return cdnAggregated[i].LagCount > cdnAggregated[j].LagCount
	})

	// 生成Client聚合数据
	var clientAggregated []AggregatedData
	var aggData AggData
	clientCountryCntMap := make(map[string]int)
	areaCntMap := make(map[string]int)
	provCntMap := make(map[string]int)
	totalLagCnt := 0
	for clientIp, total := range clientTotalCntMap {
		cdnIps := make([]string, 0)
		for cdnip := range clientCdnIpsMap[clientIp] {
			cdnIps = append(cdnIps, cdnip)
		}
		country, isp, area, prov := util.GetLocate(clientIp, ipparser)
		clientAggregated = append(clientAggregated, AggregatedData{
			IP:         clientIp,
			TotalCount: total,
			LagCount:   clientLagCntMap[clientIp],
			Prov:       prov,
			Isp:        isp,
			RemoteIps:  cdnIps,
		})
		totalLagCnt += clientLagCntMap[clientIp]
		clientCountryCntMap[country]++
		areaCntMap[area]++
		provCntMap[prov]++
	}
	for i := range clientAggregated {
		clientAggregated[i].LagRate = float64(clientAggregated[i].LagCount*100) / float64(totalLagCnt)
	}
	aggData.CountryCntMap = clientCountryCntMap
	aggData.AreaCntMap = areaCntMap
	aggData.ProvCntMap = provCntMap
	aggData.AreaLagCntMap = areaLagCntMap
	aggData.ProvLagCntMap = provLagCntMap

	// 按不良质量次数降序排序
	sort.Slice(clientAggregated, func(i, j int) bool {
		return clientAggregated[i].LagCount > clientAggregated[j].LagCount
	})

	var cdnAggregateData []CdnAggregateData
	for cdnip, clientsMap := range cdnLagClientMap {
		var lagIps []string
		for clientIp := range clientsMap {
			lagIps = append(lagIps, clientIp)
		}
		cdnAggregateData = append(cdnAggregateData, CdnAggregateData{
			IP:             cdnip,
			TotalUserCount: len(cdnAllClientMap[cdnip]),
			LagIps:         lagIps,
		})
	}

	return cdnAggregated, clientAggregated, cdnAggregateData, aggData
}

// OnlineUserAggregatedData 在线用户聚合数据结构
type OnlineUserAggregatedData struct {
	Timestamp string `json:"timestamp"`
	OnlineNum int    `json:"online_num"`
}

// aggregateOnlineUsers 按分钟聚合在线用户数（模拟指定SQL查询的效果）
func aggregateOnlineUsers(reports []util.QualityReport) []OnlineUserAggregatedData {
	// 按分钟分组聚合
	minuteMap := make(map[string]map[string]bool) // timestamp -> set of distinct IPs

	for _, report := range reports {
		// 检查是否满足筛选条件

		// 1. 检查日期和小时（从cts时间戳解析）
		var timestamp time.Time
		if report.Cts != nil {
			timestamp = time.Unix(*report.Cts, 0).In(time.FixedZone("Asia/Shanghai", 8*60*60))
		} else {
			continue
		}

		// 2. 检查流URL条件
		streamUrl := ""
		if report.DimStreamUrl != nil {
			streamUrl = *report.DimStreamUrl
		}
		if streamUrl == "" {
			continue
		}

		// 4. 检查ISP条件
		isp := ""
		if report.DimIsp != nil {
			isp = *report.DimIsp
		}
		if !strings.Contains(strings.ToLower(isp), "china") {
			continue
		}

		// 5. 检查IP
		ip := ""
		if report.DimIp != nil {
			ip = *report.DimIp
		}
		if ip == "" {
			continue
		}

		// 按分钟分组，使用date_trunc('minute')的效果
		minuteTimestamp := time.Date(
			timestamp.Year(), timestamp.Month(), timestamp.Day(),
			timestamp.Hour(), timestamp.Minute(), 0, 0,
			time.FixedZone("Asia/Shanghai", 8*60*60),
		).Format("2006-01-02 15:04:05")

		// 初始化分钟IP集合
		if _, exists := minuteMap[minuteTimestamp]; !exists {
			minuteMap[minuteTimestamp] = make(map[string]bool)
		}

		// 添加IP到集合中（去重）
		minuteMap[minuteTimestamp][ip] = true
	}

	// 转换为结果数组
	var result []OnlineUserAggregatedData
	for timestamp, ipSet := range minuteMap {
		result = append(result, OnlineUserAggregatedData{
			Timestamp: timestamp,
			OnlineNum: len(ipSet), // 去重后的IP数量
		})
	}

	// 按时间排序
	sort.Slice(result, func(i, j int) bool {
		timeI, _ := time.Parse("2006-01-02 15:04:05", result[i].Timestamp)
		timeJ, _ := time.Parse("2006-01-02 15:04:05", result[j].Timestamp)
		return timeI.Before(timeJ)
	})

	return result
}

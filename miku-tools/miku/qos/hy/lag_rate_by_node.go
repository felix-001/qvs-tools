package hy

import (
	"log"
	"mikutool/miku/qos"
	"mikutool/miku/qos/html"
	"mikutool/public/util"
	"net/url"
	"strings"

	"github.com/qbox/pili/common/ipdb.v1"
)

// 每个节点的卡顿率

func init() {
	qos.RegisterChartGenerator("hy_cdn_lag", &HyCdnLag{})
}

type HyCdnLag struct {
	ipparser *ipdb.City
}

func (h *HyCdnLag) getRawData(req qos.QOSRequest) []util.HyCdnLagReport {
	sql := qos.BuildHyCdnLagSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyCdnLagReports []util.HyCdnLagReport
	if err := util.TrinoQuery("miku", sql, &hyCdnLagReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return nil
	}
	log.Println("查询结果hyCdnLagReports:", len(hyCdnLagReports))
	if req.RawData {
		qos.SaveHyCdnLagRawData(hyCdnLagReports)
	}
	return hyCdnLagReports
}

func (h *HyCdnLag) getClientIpsOnCdnIp(req qos.QOSRequest) ([]util.HyClientIpsOnCdnIpReport, error) {
	hyClientIpsOnCdnIpReports, err := qos.GetHyDistinctCdnClientIps(req)
	log.Println("查询结果hyClientIpsOnCdnIpReports:", len(hyClientIpsOnCdnIpReports))
	if req.RawData {
		qos.SaveHyClientIpsOnCdnIpRawData(hyClientIpsOnCdnIpReports)
	}
	return hyClientIpsOnCdnIpReports, err
}

func (h *HyCdnLag) getStreamIdFromUrl(addr string) string {
	if addr == "" {
		return ""
	}

	// 解析URL
	u, err := url.Parse(addr)
	if err != nil {
		log.Printf("解析URL失败: %s, 错误: %v", addr, err)
		return ""
	}

	// 提取路径部分
	path := u.Path
	if path == "" {
		return ""
	}

	// 支持的扩展名列表
	supportedExtensions := []string{".flv", ".m3u8", ".slice", ".pstream"}

	// 查找最后一个支持的扩展名
	var streamId string
	var foundExt string
	for _, ext := range supportedExtensions {
		if idx := strings.LastIndex(path, ext); idx != -1 {
			// 找到扩展名，提取从前面最后一个'/'到扩展名结束的部分
			startIdx := strings.LastIndex(path[:idx], "/")
			if startIdx != -1 {
				streamId = path[startIdx+1 : idx+len(ext)]
				foundExt = ext
				break
			}
		}
	}

	if streamId == "" {
		// 如果没有找到支持的扩展名，返回空字符串
		return ""
	}

	// 获取query参数
	query := u.Query()

	// 构建最终的stream id
	result := streamId

	// 检查codec参数
	if codec := query.Get("codec"); codec != "" {
		// 在扩展名前面插入_codec
		extIdx := strings.LastIndex(result, foundExt)
		if extIdx != -1 {
			result = result[:extIdx] + "_" + codec + result[extIdx:]
		}
	}

	// 检查ratio参数
	if ratio := query.Get("ratio"); ratio != "" {
		// 在扩展名前面插入_ratio
		extIdx := strings.LastIndex(result, foundExt)
		if extIdx != -1 {
			result = result[:extIdx] + "_" + ratio + result[extIdx:]
		}
	}

	return result
}

func (h *HyCdnLag) aggCdnLagData(hyCdnLagReports []util.HyCdnLagReport, clientIpsOnCdnIpReports []util.HyClientIpsOnCdnIpReport) []qos.AggregatedData {
	totalLagCnt := 0
	for _, report := range hyCdnLagReports {
		if report.LagCnt == nil || report.DimCdnip == nil || report.Total == nil {
			continue
		}
		totalLagCnt += *report.LagCnt
	}

	// cdn ip ->  client ips
	var cdnIp2NormalClientipsMap = make(map[string]map[string]bool)
	var cdnIp2LagClientIpsMap = make(map[string]map[string]bool)
	var cdnIp2NormalStreamsMap = make(map[string]map[string]bool)
	var cdnIp2LagStreamsMap = make(map[string]map[string]bool)
	for _, report := range clientIpsOnCdnIpReports {
		if report.DimCdnip == nil || report.DimIp == nil {
			continue
		}
		if _, ok := cdnIp2NormalClientipsMap[*report.DimCdnip]; !ok {
			cdnIp2NormalClientipsMap[*report.DimCdnip] = make(map[string]bool)
		}
		if _, ok := cdnIp2LagClientIpsMap[*report.DimCdnip]; !ok {
			cdnIp2LagClientIpsMap[*report.DimCdnip] = make(map[string]bool)
		}
		if _, ok := cdnIp2NormalStreamsMap[*report.DimCdnip]; !ok {
			cdnIp2NormalStreamsMap[*report.DimCdnip] = make(map[string]bool)
		}
		if _, ok := cdnIp2LagStreamsMap[*report.DimCdnip]; !ok {
			cdnIp2LagStreamsMap[*report.DimCdnip] = make(map[string]bool)
		}
		streamid := h.getStreamIdFromUrl(*report.DimStreamUrl)
		if *report.LagCnt == 0 {
			cdnIp2NormalClientipsMap[*report.DimCdnip][*report.DimIp] = true
			cdnIp2NormalStreamsMap[*report.DimCdnip][streamid] = true
		} else {
			cdnIp2LagClientIpsMap[*report.DimCdnip][*report.DimIp] = true
			cdnIp2LagStreamsMap[*report.DimCdnip][streamid] = true
		}
	}

	var aggCdnLagDatas []qos.AggregatedData
	for _, report := range hyCdnLagReports {
		if report.LagCnt == nil || report.DimCdnip == nil || report.Total == nil {
			continue
		}
		cdnIp := *report.DimCdnip
		var normalClientIps []string
		if _, ok := cdnIp2NormalClientipsMap[cdnIp]; ok {
			for clientIp := range cdnIp2NormalClientipsMap[cdnIp] {
				normalClientIps = append(normalClientIps, clientIp)
			}
		}
		var lagClientIps []string
		if _, ok := cdnIp2LagClientIpsMap[cdnIp]; ok {
			for clientIp := range cdnIp2LagClientIpsMap[cdnIp] {
				lagClientIps = append(lagClientIps, clientIp)
			}
		}
		var normalStreams []string
		if _, ok := cdnIp2NormalStreamsMap[cdnIp]; ok {
			for stream := range cdnIp2NormalStreamsMap[cdnIp] {
				normalStreams = append(normalStreams, stream)
			}
		}
		var lagStreams []string
		if _, ok := cdnIp2LagStreamsMap[cdnIp]; ok {
			for stream := range cdnIp2LagStreamsMap[cdnIp] {
				lagStreams = append(lagStreams, stream)
			}
		}
		_, isp, _, prov := util.GetLocate(cdnIp, h.ipparser)
		aggCdnLagDatas = append(aggCdnLagDatas, qos.AggregatedData{
			IP:            cdnIp,
			LagCount:      *report.LagCnt,
			TotalCount:    *report.Total,
			Isp:           isp,
			Prov:          prov,
			NormalIps:     normalClientIps,
			LagIps:        lagClientIps,
			LagRate:       float64(*report.LagCnt*100) / float64(totalLagCnt),
			LagUsrCnt:     len(lagClientIps),
			TotalUsrCnt:   len(lagClientIps) + len(normalClientIps),
			LagUsrRate:    float64(len(lagClientIps)*100) / float64(len(lagClientIps)+len(normalClientIps)),
			NormalStreams: normalStreams,
			LagStreams:    lagStreams,
		})
	}

	return aggCdnLagDatas
}

func (h *HyCdnLag) Generate(req qos.QOSRequest) string {
	h.ipparser = req.IpParser
	hyCdnLagReports := h.getRawData(req)
	clientIpsOnCdnIpReports, err := h.getClientIpsOnCdnIp(req)
	if err != nil {
		log.Printf("获取客户端IP聚合数据失败: %v", err)
		return ""
	}
	aggCdnLagData := h.aggCdnLagData(hyCdnLagReports, clientIpsOnCdnIpReports)
	//log.Println("aggCdnLagData:", aggCdnLagData)
	tableHTML := qos.GenerateTableHTML(aggCdnLagData, []qos.AggregatedData{})

	table := html.NewTable()
	rowDatas := `
		[
			{ make: "Tesla", model: "Model Y", price: 64950, electric: true },
			{ make: "Ford", model: "F-Series", price: 33850, electric: false },
			{ make: "Toyota", model: "Corolla", price: 29600, electric: false },
			{ make: "Mercedes", model: "EQA", price: 48890, electric: true },
			{ make: "Fiat", model: "500", price: 15774, electric: false },
			{ make: "Nissan", model: "Juke", price: 20675, electric: false },
		]
	`
	table.SetRowDatas(rowDatas)
	columns := `
		[
			{ field: "make" },
			{ field: "model" },
			{ field: "price" },
			{ field: "electric" },
		]
	`
	table.SetColumns(columns)
	return tableHTML + table.String()
}

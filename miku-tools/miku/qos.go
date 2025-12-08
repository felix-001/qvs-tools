package miku

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
)

// QOSRequest 前端查询请求结构
type QOSRequest struct {
	AppName     string `json:"appName"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	StreamID    string `json:"streamId"`
	Domain      string `json:"domain"`
	UID         string `json:"uid"`
	Hour        string `json:"hour"`
	FuzzySearch bool   `json:"fuzzySearch"`
	LogLevel    string `json:"logLevel"`
	RawData     bool   `json:"rawData"`
}

// QOSServer HTTP服务器结构
type QOSServer struct {
	config    *config.Config
	resources *resources.Resources
}

// NewQOSServer 创建新的QOS服务器
func NewQOSServer(cfg *config.Config, resources *resources.Resources) *QOSServer {
	return &QOSServer{
		config:    cfg,
		resources: resources,
	}
}

// StartServer 启动HTTP服务器
func (s *QOSServer) StartServer() {
	// 静态文件服务
	http.HandleFunc("/", s.homeHandler)

	// API接口
	http.HandleFunc("/api/v1/appnames", s.getAppNamesHandler)
	http.HandleFunc("/api/v1/qos", s.qosAnalysisHandler)

	fmt.Println("QOS排障系统启动成功！")
	fmt.Println("请访问: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// homeHandler 主页面处理器
func (s *QOSServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// 返回HTML页面
	tmpl := template.Must(template.New("home").Parse(s.getHomePageTemplate()))
	tmpl.Execute(w, nil)
}

// getAppNamesHandler 获取AppName列表处理器
func (s *QOSServer) getAppNamesHandler(w http.ResponseWriter, r *http.Request) {
	// 模拟AppName列表，实际应该从数据库获取
	appNames := []string{
		"huyacdn",
		"livessports",
		"douyu",
		"vzan",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appNames)
}

// qosAnalysisHandler QOS分析处理器
func (s *QOSServer) qosAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QOSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check and adjust time format if needed
	if strings.Count(req.StartTime, ":") == 1 {
		req.StartTime = req.StartTime + ":00"
	}
	if strings.Count(req.EndTime, ":") == 1 {
		req.EndTime = req.EndTime + ":00"
	}

	// 构建SQL查询（获取原始数据）
	sql := s.buildSQLQuery(req, true)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}

	// 执行查询
	var reports []util.QualityReport
	if err := util.TrinoQuery("miku", sql, &reports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果:", len(reports))
	if req.RawData {
		s.saveCsv(reports)
	}

	var streamdReports []util.StreamdLagReport
	sql = s.buildMikuSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	if err := util.TrinoQuery("miku", sql, &streamdReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果streamdReports:", len(streamdReports))

	var streamdFpsReports []util.StreamdFpsReport
	var streamUpstreamBandwidthReports []util.StreamdUpstreamBandWidthReport
	if req.StreamID != "" && !req.FuzzySearch {
		sql = s.buildMikuFpsSQLQuery(req)
		if req.LogLevel == "detail" {
			log.Printf("执行SQL查询: %s", sql)
		}
		if err := util.TrinoQuery("miku", sql, &streamdFpsReports); err != nil {
			log.Printf("Trino查询失败: %v", err)
			http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Println("查询结果streamdFpsReports:", len(streamdFpsReports))

		sql = s.buildMikuUpstreamBandwidthSQLQuery(req)
		if req.LogLevel == "detail" {
			log.Printf("执行SQL查询: %s", sql)
		}
		if err := util.TrinoQuery("miku", sql, &streamUpstreamBandwidthReports); err != nil {
			log.Printf("Trino查询失败: %v", err)
			http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Println("查询结果streamUpstreamBandwidthReport:", len(streamUpstreamBandwidthReports))
	}

	sql = s.buildMikuStreamCntSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var streamCntReports []util.StreamdStreamCntReport
	if err := util.TrinoQuery("miku", sql, &streamCntReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果streamCntReports:", len(streamCntReports))

	sql = s.buidUpstreamDistributeSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var upstreamDistributeReports []util.UpstreamDistributeReport
	if err := util.TrinoQuery("miku", sql, &upstreamDistributeReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果upstreamDistributeReports:", len(upstreamDistributeReports))

	// 在Go代码中实现按分钟聚合（模拟第二个SQL查询的效果）
	minuteAggregated := s.aggregateByMinute(reports)

	cdnAggregated, clientAggregated, cdnAggData, aggData := s.aggregateReport(reports)

	// 聚合在线用户数（使用示例日期和小时，实际应该从请求参数获取）
	onlineUsersAggregated := s.aggregateOnlineUsers(reports)

	streamdChartsHTML := s.generateStreamdChartsHTML(streamdReports)

	// 生成推流/回源帧率折线图
	streamdVideoFpsChartHTML := s.generateStreamdVideoFpsChartHTML(streamdFpsReports)
	streamdAudioFpsChartHTML := s.generateStreamdAudioFpsChartHTML(streamdFpsReports)

	// 生成在线流个数折线图
	streamCntChartHTML := s.generateStreamCntChartHTML(streamCntReports)

	// 生成推流/回源带宽折线图
	upstreamBandwidthChartHTML := s.generateUpstreamBandwidthChartHTML(streamUpstreamBandwidthReports)

	areaCntMap, provCntMap := s.aggUpstreamDistributeReports(upstreamDistributeReports)

	// 生成源站分布饼图
	upstreamDistributeChartsHTML := s.generateUpstreamDistributeChartsHTML(areaCntMap, provCntMap)

	// 生成分钟聚合数据的折线图
	minuteChartHTML := s.generateMinuteChartHTML(minuteAggregated)

	// 生成卡顿用户占比折线图
	lagUserRatioChartHTML := s.generateLagUserRatioChartHTML(minuteAggregated)

	// 生成节点卡顿占比折线图
	cdnLagRatioChartHTML := s.generateCdnLagRatioChartHTML(minuteAggregated)

	// 生成在线用户数折线图
	onlineUsersChartHTML := s.generateOnlineUsersChartHTML(onlineUsersAggregated)

	// 生成聚合数据表格HTML
	tableHTML := s.generateTableHTML(cdnAggregated, clientAggregated)

	// 生成CDN卡顿用户表格HTML
	cdnLagTableHTML := s.generateCDNLagTableHTML(cdnAggData)

	// 生成聚合数据饼图HTML
	aggDataChartsHTML := s.generateAggDataChartsHTML(aggData)

	// 合并图表和表格HTML
	fullHTML := streamdChartsHTML + streamdVideoFpsChartHTML + streamdAudioFpsChartHTML + upstreamBandwidthChartHTML + streamCntChartHTML + upstreamDistributeChartsHTML + minuteChartHTML + lagUserRatioChartHTML + cdnLagRatioChartHTML + onlineUsersChartHTML + aggDataChartsHTML + tableHTML + cdnLagTableHTML

	// 返回完整的HTML
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fullHTML))
}

func (s *QOSServer) convertToDay(timeStr string) string {
	// 尝试解析不同格式的时间字符串
	layouts := []string{
		"2006-01-02T15:04:05", // datetime-local格式
		"2006-01-02 15:04:05", // 标准日期时间格式
		"2006-01-02",          // 仅日期格式
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			result := t.Format("20060102") // 20251203格式
			log.Printf("时间转换成功: 输入=%s, 布局=%s, 结果=%s", timeStr, layout, result)
			return result
		}
	}

	// 如果解析失败，直接返回原始字符串
	log.Printf("时间转换失败，返回原始字符串: %s", timeStr)
	return timeStr
}

// getString 安全获取字符串指针值
func (s *QOSServer) getString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

// getInt64 安全获取int64指针值
func (s *QOSServer) getInt64(i *int64) string {
	if i == nil {
		return ""
	}
	return strconv.FormatInt(*i, 10)
}

// MinuteAggregatedData 按分钟聚合数据结构
type MinuteAggregatedData struct {
	Timestamp    string               `json:"timestamp"`
	Percent      float64              `json:"percent"`
	LagCount     int                  `json:"lag_count"`
	TotalCount   int                  `json:"total_count"`
	LagUserCnt   int                  `json:"lag_user_cnt"`
	TotalUserCnt int                  `json:"total_user_cnt"`
	LagCdnCnt    int                  `json:"lag_cdn_cnt"`
	TotalCdnCnt  int                  `json:"total_cdn_cnt"`
	Reports      []util.QualityReport `json:"report"`
}

// AggregatedData 聚合数据结构
type AggregatedData struct {
	IP         string   `json:"ip"`
	TotalCount int      `json:"total_count"`
	LagCount   int      `json:"lag_count"`
	Prov       string   `json:"prov"`
	Isp        string   `json:"isp"`
	RemoteIps  []string `json:"remote_ips"`
	LagRate    float64  `json:"lag_rate"`
}

type CdnAggregateData struct {
	IP             string   `json:"ip"`
	TotalUserCount int      `json:"total_user_count"`
	LagIps         []string `json:"lag_ips"`
}

type AggData struct {
	CountryCntMap map[string]int
	AreaCntMap    map[string]int
	ProvCntMap    map[string]int
	AreaLagCntMap map[string]int
	ProvLagCntMap map[string]int
}

// aggregateByMinute 按分钟聚合数据（模拟第二个SQL查询的效果）
func (s *QOSServer) aggregateByMinute(reports []util.QualityReport) []MinuteAggregatedData {
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

func (s *QOSServer) aggregateReport(reports []util.QualityReport) ([]AggregatedData, []AggregatedData, []CdnAggregateData, AggData) {
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
				_, _, area, prov := util.GetLocate(clientIp, s.resources.IpParser)
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
		_, isp, _, prov := util.GetLocate(cdnip, s.resources.IpParser)
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
		country, isp, area, prov := util.GetLocate(clientIp, s.resources.IpParser)
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
func (s *QOSServer) aggregateOnlineUsers(reports []util.QualityReport) []OnlineUserAggregatedData {
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

func (s *QOSServer) aggUpstreamDistributeReports(reports []util.UpstreamDistributeReport) (map[string]int, map[string]int) {
	areaCntMap := make(map[string]int)
	provCntMap := make(map[string]int)
	for _, report := range reports {
		if report.RemoteAddr == nil {
			continue
		}
		_, _, area, prov := util.GetLocate(*report.RemoteAddr, s.resources.IpParser)
		areaCntMap[area]++
		provCntMap[prov]++
	}
	return areaCntMap, provCntMap
}

// Qos 主函数，启动QOS服务器
func Qos(config *config.Config, resources *resources.Resources) {
	if config == nil {
		//log.Fatal("配置不能为空")
	}

	server := NewQOSServer(config, resources)
	server.StartServer()
}

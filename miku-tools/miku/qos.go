package miku

import (
	"bytes"
	"encoding/base64"
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

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
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
}

// QOSServer HTTP服务器结构
type QOSServer struct {
	config *config.Config
}

// NewQOSServer 创建新的QOS服务器
func NewQOSServer(cfg *config.Config) *QOSServer {
	return &QOSServer{
		config: cfg,
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

	// 构建SQL查询（获取原始数据）
	sql := s.buildSQLQuery(req, true)
	log.Printf("执行SQL查询: %s", sql)

	// 执行查询
	var reports []util.QualityReport
	if err := util.TrinoQuery("miku", sql, &reports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果:", len(reports))
	s.saveCsv(reports)

	var streamdReports []util.StreamdLagReport
	sql = s.buildMikuSQLQuery(req)
	log.Printf("执行SQL查询: %s", sql)
	if err := util.TrinoQuery("miku", sql, &streamdReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		http.Error(w, fmt.Sprintf("查询失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Println("查询结果streamdReports:", len(streamdReports))

	// 在Go代码中实现按分钟聚合（模拟第二个SQL查询的效果）
	minuteAggregated := s.aggregateByMinute(reports)

	// 按照TestHy1方式聚合数据
	cdnAggregated, clientAggregated := s.aggregateByTestHy1(reports)

	// 聚合在线用户数（使用示例日期和小时，实际应该从请求参数获取）
	onlineUsersAggregated := s.aggregateOnlineUsers(reports)

	streamdChartsHTML := s.generateStreamdChartsHTML(streamdReports)

	// 生成分钟聚合数据的折线图
	minuteChartHTML := s.generateMinuteChartHTML(minuteAggregated)

	// 生成在线用户数折线图
	onlineUsersChartHTML := s.generateOnlineUsersChartHTML(onlineUsersAggregated)

	// 生成聚合数据表格HTML
	tableHTML := s.generateTableHTML(cdnAggregated, clientAggregated)

	// 合并图表和表格HTML
	fullHTML := streamdChartsHTML + minuteChartHTML + onlineUsersChartHTML + tableHTML

	// 返回完整的HTML
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fullHTML))
}

// convertToDay 将时间字符串转换为day格式 (YYYYMMDD)
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
	Timestamp  string  `json:"timestamp"`
	Percent    float64 `json:"percent"`
	LagCount   int     `json:"lag_count"`
	TotalCount int     `json:"total_count"`
}

// AggregatedData 聚合数据结构
type AggregatedData struct {
	CDNIP      string  `json:"cdn_ip"`
	BadCount   int     `json:"bad_count"`
	TotalCount int     `json:"total_count"`
	BadPercent float64 `json:"bad_percent"`
	LagCount   int     `json:"lag_count"`
	NoLagCount int     `json:"no_lag_count"`
}

// ClientAggregatedData Client聚合数据
type ClientAggregatedData struct {
	ClientIP   string  `json:"client_ip"`
	BadCount   int     `json:"bad_count"`
	TotalCount int     `json:"total_count"`
	BadPercent float64 `json:"bad_percent"`
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

		// 检查是否为不良质量
		if report.FieldVideoBadQuality != nil && *report.FieldVideoBadQuality == 100 {
			minuteData.LagCount++
		}
	}

	// 计算百分比并转换为切片
	var result []MinuteAggregatedData
	for _, data := range minuteMap {
		if data.TotalCount > 0 {
			data.Percent = float64(data.LagCount) / float64(data.TotalCount) * 100
		}
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

// aggregateByTestHy1 按照TestHy1方式聚合数据
func (s *QOSServer) aggregateByTestHy1(reports []util.QualityReport) ([]AggregatedData, []ClientAggregatedData) {
	// CDN IP聚合
	cdnBadMap := make(map[string]bool)   // cdnip -> field_video_bad_quality=="100"
	cdnAllMap := make(map[string]bool)   // 所有cdnip
	cdnLagCnt := make(map[string]int)    // cdnip -> 延迟数
	cdnNogLagCnt := make(map[string]int) // cdnip -> 无延迟数

	// Client IP聚合
	clientBadMap := make(map[string]bool) // clientIp -> field_video_bad_quality=="100"
	clientAllMap := make(map[string]bool) // 所有clientIp

	for _, report := range reports {
		// 获取CDN IP
		var cdnip string
		if report.DimCdnip != nil && *report.DimCdnip != "" {
			cdnip = *report.DimCdnip
		}

		if (cdnip == "" || cdnip == "qn.flv.huya.com" || cdnip == "http://qn.flv.huya.com") && report.DimStreamUrl != nil && *report.DimStreamUrl != "" {
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
			cdnAllMap[cdnip] = true
			if isBadQuality {
				cdnBadMap[cdnip] = true
				cdnLagCnt[cdnip]++
			} else {
				cdnNogLagCnt[cdnip]++
			}
		}

		// 处理Client IP聚合
		if clientIp != "" {
			clientAllMap[clientIp] = true
			if isBadQuality {
				clientBadMap[clientIp] = true
			}
		}
	}

	// 生成CDN聚合数据
	var cdnAggregated []AggregatedData
	for cdnip := range cdnAllMap {
		badCount := 0
		if cdnBadMap[cdnip] {
			badCount = 1
		}
		totalCount := 1
		if cdnLagCnt[cdnip] > 0 || cdnNogLagCnt[cdnip] > 0 {
			totalCount = cdnLagCnt[cdnip] + cdnNogLagCnt[cdnip]
		}

		badPercent := 0.0
		if totalCount > 0 {
			badPercent = float64(badCount) / float64(totalCount) * 100
		}

		cdnAggregated = append(cdnAggregated, AggregatedData{
			CDNIP:      cdnip,
			BadCount:   badCount,
			TotalCount: totalCount,
			BadPercent: badPercent,
			LagCount:   cdnLagCnt[cdnip],
			NoLagCount: cdnNogLagCnt[cdnip],
		})
	}

	// 按延迟次数降序排序
	sort.Slice(cdnAggregated, func(i, j int) bool {
		return cdnAggregated[i].LagCount > cdnAggregated[j].LagCount
	})

	// 生成Client聚合数据
	var clientAggregated []ClientAggregatedData
	for clientIp := range clientAllMap {
		badCount := 0
		if clientBadMap[clientIp] {
			badCount = 1
		}
		totalCount := 1

		badPercent := 0.0
		if totalCount > 0 {
			badPercent = float64(badCount) / float64(totalCount) * 100
		}

		clientAggregated = append(clientAggregated, ClientAggregatedData{
			ClientIP:   clientIp,
			BadCount:   badCount,
			TotalCount: totalCount,
			BadPercent: badPercent,
		})
	}

	// 按不良质量次数降序排序
	sort.Slice(clientAggregated, func(i, j int) bool {
		return clientAggregated[i].BadCount > clientAggregated[j].BadCount
	})

	return cdnAggregated, clientAggregated
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

// generateOnlineUsersChartHTML 生成在线用户数折线图HTML
func (s *QOSServer) generateOnlineUsersChartHTML(onlineUsersAggregated []OnlineUserAggregatedData) string {
	if len(onlineUsersAggregated) == 0 {
		log.Println("generateOnlineUsersChartHTML: no data")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "每分钟在线用户数趋势",
			Left:  "right",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			Formatter: "{a} <br/>{b} : {c}",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
			AxisLabel: &opts.AxisLabel{
				Rotate:   45,
				Interval: "auto",
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type: "value",
			AxisLabel: &opts.AxisLabel{
				Formatter: "{value}",
			},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "100%",
			Height: "400px",
			Theme:  "white",
		}),
	)

	// 准备X轴数据（时间戳）
	var xAxisData []string
	// 准备Y轴数据（在线用户数）
	var yAxisData []opts.LineData

	for _, data := range onlineUsersAggregated {
		xAxisData = append(xAxisData, data.Timestamp)
		yAxisData = append(yAxisData, opts.LineData{
			Value: data.OnlineNum,
		})
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("在线用户数", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(false),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#52c41a",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#52c41a",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成在线用户图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 返回包含iframe的HTML
	return fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">每分钟在线用户数趋势图</h2>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 450px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, encodedHTML)
}

// Qos 主函数，启动QOS服务器
func Qos(config *config.Config) {
	if config == nil {
		//log.Fatal("配置不能为空")
	}

	server := NewQOSServer(config)
	server.StartServer()
}

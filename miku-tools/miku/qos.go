package miku

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mikutool/config"
	"mikutool/public/util"
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
		"miku_live",
		"miku_vod",
		"miku_short",
		"miku_game",
		"miku_education",
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

	// 构建SQL查询
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

	// 生成图表HTML
	chartHTML := s.generateChartHTML(reports)

	// 返回图表HTML
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(chartHTML))
}

// buildSQLQuery 构建SQL查询语句
func (s *QOSServer) buildSQLQuery(req QOSRequest, raw bool) string {
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
		startDay := s.convertToDay(req.StartTime)
		log.Printf("StartTime输入: %s, 转换后: %s", req.StartTime, startDay)
		sql += fmt.Sprintf(" AND day >= '%s'", startDay)
	}

	if req.EndTime != "" {
		endDay := s.convertToDay(req.EndTime)
		log.Printf("EndTime输入: %s, 转换后: %s", req.EndTime, endDay)
		sql += fmt.Sprintf(" AND day <= '%s'", endDay)
	}

	// 添加流ID过滤
	if req.StreamID != "" {
		if req.FuzzySearch {
			sql += fmt.Sprintf(" AND dim_stream LIKE '%%%s%%'", req.StreamID)
		} else {
			sql += fmt.Sprintf(" AND dim_stream = '%s'", req.StreamID)
		}
	}

	// 添加域名过滤
	if req.Domain != "" {
		if req.FuzzySearch {
			sql += fmt.Sprintf(" AND dim_cdndomain LIKE '%%%s%%'", req.Domain)
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
	sql += " LIMIT 10000"

	return sql
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

// generateChartHTML 生成图表HTML
func (s *QOSServer) generateChartHTML(reports []util.QualityReport) string {
	if len(reports) == 0 {
		return `
		<html>
		<head><title>QOS分析结果</title></head>
		<body>
			<h2>QOS分析结果</h2>
			<p>未找到匹配的数据</p>
			<button onclick="window.history.back()">返回</button>
		</body>
		</html>
		`
	}

	// 使用echarts生成可视化图表
	html := `
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>QOS分析结果</title>
		<script src="https://cdn.jsdelivr.net/npm/echarts@5.4.3/dist/echarts.min.js"></script>
		<style>
			body { 
				font-family: 'Microsoft YaHei', Arial, sans-serif; 
				margin: 20px; 
				background-color: #f5f5f5;
			}
			.container {
				max-width: 1400px;
				margin: 0 auto;
				background-color: white;
				padding: 30px;
				border-radius: 10px;
				box-shadow: 0 2px 10px rgba(0,0,0,0.1);
			}
			h1 { 
				color: #333; 
				text-align: center;
				margin-bottom: 30px;
				border-bottom: 3px solid #007bff;
				padding-bottom: 10px;
			}
			.chart-container {
				display: grid;
				grid-template-columns: 1fr 1fr;
				gap: 20px;
				margin-bottom: 30px;
			}
			.chart-box {
				border: 1px solid #ddd;
				border-radius: 8px;
				padding: 15px;
				background-color: white;
				box-shadow: 0 1px 3px rgba(0,0,0,0.1);
			}
			.chart-title {
				font-size: 16px;
				font-weight: bold;
				margin-bottom: 15px;
				color: #333;
				text-align: center;
			}
			.chart {
				width: 100%;
				height: 300px;
			}
			.data-table {
				margin-top: 30px;
			}
			table { 
				width: 100%; 
				border-collapse: collapse; 
				margin-top: 20px;
				font-size: 12px;
			}
			th, td { 
				border: 1px solid #ddd; 
				padding: 8px; 
				text-align: left;
				max-width: 150px;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}
			th { 
				background-color: #f2f2f2; 
				font-weight: bold;
			}
			.back-btn { 
				background-color: #007bff; 
				color: white; 
				padding: 10px 20px; 
				border: none; 
				border-radius: 4px; 
				cursor: pointer;
				margin-bottom: 20px;
				font-size: 14px;
				transition: background-color 0.3s;
			}
			.back-btn:hover {
				background-color: #0056b3;
			}
			.summary-info {
				display: grid;
				grid-template-columns: repeat(4, 1fr);
				gap: 20px;
				margin-bottom: 30px;
			}
			summary-card {
				background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
				color: white;
				padding: 20px;
				border-radius: 8px;
				text-align: center;
			}
			summary-card h3 {
				margin: 0 0 10px 0;
				font-size: 14px;
			}
			summary-card p {
				margin: 0;
				font-size: 24px;
				font-weight: bold;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<button class="back-btn" onclick="window.history.back()">← 返回</button>
			<h1>QOS质量分析报告</h1>
			
			<!-- 统计摘要 -->
			<div class="summary-info">
				<div class="summary-card">
					<h3>总记录数</h3>
					<p>` + fmt.Sprintf("%d", len(reports)) + `</p>
				</div>
				<div class="summary-card">
					<h3>平均不良质量</h3>
					<p>` + s.calculateAverageBadQuality(reports) + `</p>
				</div>
				<div class="summary-card">
					<h3>最高码率</h3>
					<p>` + s.getMaxBitrate(reports) + `</p>
				</div>
				<div class="summary-card">
					<h3>涉及域名</h3>
					<p>` + s.getUniqueDomainCount(reports) + `</p>
				</div>
			</div>

			<!-- 图表区域 -->
			<div class="chart-container">
				<div class="chart-box">
					<div class="chart-title">不良质量趋势</div>
					<div id="qualityChart" class="chart"></div>
				</div>
				<div class="chart-box">
					<div class="chart-title">CDN域名分布</div>
					<div id="domainChart" class="chart"></div>
				</div>
				<div class="chart-box">
					<div class="chart-title">平台分布</div>
					<div id="platformChart" class="chart"></div>
				</div>
				<div class="chart-box">
					<div class="chart-title">码率分布</div>
					<div id="bitrateChart" class="chart"></div>
				</div>
			</div>

			<!-- 数据表格 -->
			<div class="data-table">
				<h2>详细数据</h2>
				<table>
					<thead>
						<tr>
							<th>时间</th>
							<th>流ID</th>
							<th>平台</th>
							<th>IP</th>
							<th>ISP</th>
							<th>CDN域名</th>
							<th>码率</th>
							<th>不良质量</th>
						</tr>
					</thead>
					<tbody>
	`

	for _, report := range reports {
		html += `
						<tr>
							<td>` + s.formatTime(report.LogTime) + `</td>
							<td>` + s.getString(report.DimStream) + `</td>
							<td>` + s.getString(report.DimPlatform) + `</td>
							<td>` + s.getString(report.DimIp) + `</td>
							<td>` + s.getString(report.DimIsp) + `</td>
							<td>` + s.getString(report.DimCdndomain) + `</td>
							<td>` + s.getString(report.DimCoderatebps) + `</td>
							<td>` + s.getInt64(report.FieldVideoBadQuality) + `</td>
						</tr>
		`
	}

	html += `
					</tbody>
				</table>
			</div>
		</div>

		<script>
			// 准备图表数据
			const reports = ` + s.generateJSReports(reports) + `;
			
			// 质量趋势图表
			const qualityChart = echarts.init(document.getElementById('qualityChart'));
			const qualityOption = {
				title: { show: false },
				tooltip: { trigger: 'axis' },
				xAxis: { 
					type: 'category',
					data: reports.map(r => r.time),
					axisLabel: { rotate: 45 }
				},
				yAxis: { type: 'value' },
				series: [{
					name: '不良质量',
					type: 'line',
					data: reports.map(r => r.badQuality),
					itemStyle: { color: '#ff6b6b' },
					areaStyle: { opacity: 0.3 }
				}]
			};
			qualityChart.setOption(qualityOption);

			// CDN域名分布图表
			const domainChart = echarts.init(document.getElementById('domainChart'));
			const domainData = {};
			reports.forEach(r => {
				domainData[r.domain] = (domainData[r.domain] || 0) + 1;
			});
			const domainOption = {
				title: { show: false },
				tooltip: { trigger: 'item' },
				series: [{
					name: '域名分布',
					type: 'pie',
					radius: '60%',
					data: Object.entries(domainData).map(([name, value]) => ({ name, value })),
					emphasis: {
						itemStyle: {
							shadowBlur: 10,
							shadowOffsetX: 0,
							shadowColor: 'rgba(0, 0, 0, 0.5)'
						}
					}
				}]
			};
			domainChart.setOption(domainOption);

			// 平台分布图表
			const platformChart = echarts.init(document.getElementById('platformChart'));
			const platformData = {};
			reports.forEach(r => {
				platformData[r.platform] = (platformData[r.platform] || 0) + 1;
			});
			const platformOption = {
				title: { show: false },
				tooltip: { trigger: 'item' },
				series: [{
					name: '平台分布',
					type: 'pie',
					radius: '60%',
					data: Object.entries(platformData).map(([name, value]) => ({ name, value })),
					emphasis: {
						itemStyle: {
							shadowBlur: 10,
							shadowOffsetX: 0,
							shadowColor: 'rgba(0, 0, 0, 0.5)'
						}
					}
				}]
			};
			platformChart.setOption(platformOption);

			// 码率分布图表
			const bitrateChart = echarts.init(document.getElementById('bitrateChart'));
			const bitrateGroups = {
				'0-1Mbps': 0, '1-2Mbps': 0, '2-5Mbps': 0, 
				'5-10Mbps': 0, '10Mbps+': 0
			};
			reports.forEach(r => {
				const bitrate = parseFloat(r.bitrate) || 0;
				if (bitrate < 1) bitrateGroups['0-1Mbps']++;
				else if (bitrate < 2) bitrateGroups['1-2Mbps']++;
				else if (bitrate < 5) bitrateGroups['2-5Mbps']++;
				else if (bitrate < 10) bitrateGroups['5-10Mbps']++;
				else bitrateGroups['10Mbps+']++;
			});
			const bitrateOption = {
				title: { show: false },
				tooltip: { trigger: 'axis' },
				xAxis: { type: 'category', data: Object.keys(bitrateGroups) },
				yAxis: { type: 'value' },
				series: [{
					name: '码率分布',
					type: 'bar',
					data: Object.values(bitrateGroups),
					itemStyle: { color: '#4ecdc4' }
				}]
			};
			bitrateChart.setOption(bitrateOption);

			// 响应式处理
			window.addEventListener('resize', () => {
				qualityChart.resize();
				domainChart.resize();
				platformChart.resize();
				bitrateChart.resize();
			});
		</script>
	</body>
	</html>
	`

	return html
}

// formatTime 格式化时间戳
func (s *QOSServer) formatTime(t *int64) string {
	if t == nil {
		return ""
	}
	return time.Unix(*t, 0).Format("2006-01-02 15:04:05")
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

// calculateAverageBadQuality 计算平均不良质量值
func (s *QOSServer) calculateAverageBadQuality(reports []util.QualityReport) string {
	if len(reports) == 0 {
		return "0"
	}

	var total int64
	for _, report := range reports {
		if report.FieldVideoBadQuality != nil {
			total += *report.FieldVideoBadQuality
		}
	}

	avg := float64(total) / float64(len(reports))
	return fmt.Sprintf("%.1f", avg)
}

// getMaxBitrate 获取最高码率
func (s *QOSServer) getMaxBitrate(reports []util.QualityReport) string {
	var maxBitrate float64
	for _, report := range reports {
		if report.DimCoderatebps != nil {
			if bitrate, err := strconv.ParseFloat(*report.DimCoderatebps, 64); err == nil {
				if bitrate > maxBitrate {
					maxBitrate = bitrate
				}
			}
		}
	}

	if maxBitrate > 0 {
		return fmt.Sprintf("%.1fMbps", maxBitrate/1000000)
	}
	return "0Mbps"
}

// getUniqueDomainCount 获取唯一域名数量
func (s *QOSServer) getUniqueDomainCount(reports []util.QualityReport) string {
	domainSet := make(map[string]bool)
	for _, report := range reports {
		if report.DimCdndomain != nil && *report.DimCdndomain != "" {
			domainSet[*report.DimCdndomain] = true
		}
	}
	return strconv.Itoa(len(domainSet))
}

// generateJSReports 生成JavaScript格式的报告数据
func (s *QOSServer) generateJSReports(reports []util.QualityReport) string {
	if len(reports) == 0 {
		return "[]"
	}

	var jsData []string
	for _, report := range reports {
		item := fmt.Sprintf(`{
			time: "%s",
			badQuality: %s,
			domain: "%s",
			platform: "%s",
			bitrate: "%s"
		}`,
			s.formatTime(report.LogTime),
			s.getInt64(report.FieldVideoBadQuality),
			s.getString(report.DimCdndomain),
			s.getString(report.DimPlatform),
			s.getString(report.DimCoderatebps))
		jsData = append(jsData, item)
	}

	return "[" + strings.Join(jsData, ",") + "]"
}

// getHomePageTemplate 获取主页HTML模板
func (s *QOSServer) getHomePageTemplate() string {
	return `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MIKU QOS排障</title>
    <style>
        body {
            font-family: 'Microsoft YaHei', Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            text-align: center;
            margin-bottom: 30px;
            border-bottom: 3px solid #007bff;
            padding-bottom: 10px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
            color: #555;
        }
        input[type="text"], select {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 14px;
            box-sizing: border-box;
        }
        .checkbox-group {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .checkbox-group input[type="checkbox"] {
            width: auto;
            margin: 0;
        }
        .datetime-group {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
        }
        .datetime-btn {
            background-color: #6c757d;
            color: white;
            padding: 8px 15px;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 12px;
            margin-top: 5px;
            transition: background-color 0.3s;
        }
        .datetime-btn:hover {
            background-color: #5a6268;
        }
        .datetime-modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0,0,0,0.5);
        }
        .datetime-modal-content {
            background-color: white;
            margin: 10% auto;
            padding: 20px;
            border-radius: 10px;
            width: 350px;
            box-shadow: 0 4px 8px rgba(0,0,0,0.2);
        }
        .datetime-modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 1px solid #eee;
        }
        .datetime-inputs {
            display: grid;
            gap: 15px;
            margin-bottom: 20px;
        }
        .datetime-input-group {
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .datetime-input-group label {
            width: 60px;
            margin: 0;
        }
        .datetime-input-group input, .datetime-input-group select {
            flex: 1;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        .modal-buttons {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
        }
        .modal-btn {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
        }
        .modal-confirm {
            background-color: #007bff;
            color: white;
        }
        .modal-cancel {
            background-color: #6c757d;
            color: white;
        }
        .submit-btn {
            background-color: #007bff;
            color: white;
            padding: 12px 30px;
            border: none;
            border-radius: 5px;
            font-size: 16px;
            cursor: pointer;
            display: block;
            margin: 30px auto 0;
            transition: background-color 0.3s;
        }
        .submit-btn:hover {
            background-color: #0056b3;
        }
        .submit-btn:disabled {
            background-color: #ccc;
            cursor: not-allowed;
        }
        .loading {
            display: none;
            text-align: center;
            margin: 20px 0;
        }
        .results {
            margin-top: 30px;
            border-top: 2px solid #eee;
            padding-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>MIKU QOS排障系统</h1>
        
        <form id="qosForm">
            <!-- 时间范围选择 -->
            <div class="datetime-group">
                <div class="form-group">
                    <label for="startTime">开始日期时间:</label>
                    <input type="text" id="startTime" name="startTime" required readonly>
                    <button type="button" class="datetime-btn" onclick="openDateTimePicker('startTime')">
                        📅 选择日期
                    </button>
                </div>
                <div class="form-group">
                    <label for="endTime">结束日期时间:</label>
                    <input type="text" id="endTime" name="endTime" required readonly>
                    <button type="button" class="datetime-btn" onclick="openDateTimePicker('endTime')">
                        📅 选择日期
                    </button>
                </div>
            </div>

            <!-- AppName选择 -->
            <div class="form-group">
                <label for="appName">应用名称:</label>
                <select id="appName" name="appName" required>
                    <option value="">请选择应用</option>
                </select>
            </div>

            <!-- 流ID输入 -->
            <div class="form-group">
                <label for="streamId">流ID:</label>
                <input type="text" id="streamId" name="streamId" placeholder="请输入流ID（可选）">
            </div>

            <!-- 域名输入 -->
            <div class="form-group">
                <label for="domain">域名:</label>
                <input type="text" id="domain" name="domain" placeholder="请输入域名（可选）">
            </div>

            <!-- UID输入 -->
            <div class="form-group">
                <label for="uid">UID:</label>
                <input type="text" id="uid" name="uid" placeholder="请输入UID（可选）">
            </div>

            <!-- 小时输入 -->
            <div class="form-group">
                <label for="hour">小时 (0-23):</label>
                <input type="number" id="hour" name="hour" min="0" max="23" placeholder="请输入小时（可选，如：14）">
            </div>

            <!-- 模糊搜索选项 -->
            <div class="form-group">
                <div class="checkbox-group">
                    <input type="checkbox" id="fuzzySearch" name="fuzzySearch">
                    <label for="fuzzySearch" style="margin: 0; font-weight: normal;">支持模糊查找</label>
                </div>
            </div>

            <!-- 提交按钮 -->
            <button type="submit" class="submit-btn">开始质量分析</button>
        </form>

        <!-- 加载提示 -->
        <div id="loading" class="loading">
            <p>正在分析数据，请稍候...</p>
        </div>

        <!-- 结果显示区域 -->
        <div id="results" class="results"></div>
    </div>

    <!-- 日期时间选择模态框 -->
    <div id="datetimeModal" class="datetime-modal">
        <div class="datetime-modal-content">
            <div class="datetime-modal-header">
                <h3>选择日期时间</h3>
                <span class="modal-btn modal-cancel" onclick="closeDateTimePicker()">✕</span>
            </div>
            <div class="datetime-inputs">
                <div class="datetime-input-group">
                    <label>日期:</label>
                    <input type="date" id="modalDate">
                </div>
                <div class="datetime-input-group">
                    <label>小时:</label>
                    <select id="modalHour">
                        <option value="00">00</option>
                        <option value="01">01</option>
                        <option value="02">02</option>
                        <option value="03">03</option>
                        <option value="04">04</option>
                        <option value="05">05</option>
                        <option value="06">06</option>
                        <option value="07">07</option>
                        <option value="08">08</option>
                        <option value="09">09</option>
                        <option value="10">10</option>
                        <option value="11">11</option>
                        <option value="12">12</option>
                        <option value="13">13</option>
                        <option value="14">14</option>
                        <option value="15">15</option>
                        <option value="16">16</option>
                        <option value="17">17</option>
                        <option value="18">18</option>
                        <option value="19">19</option>
                        <option value="20">20</option>
                        <option value="21">21</option>
                        <option value="22">22</option>
                        <option value="23">23</option>
                    </select>
                </div>
                <div class="datetime-input-group">
                    <label>分钟:</label>
                    <select id="modalMinute">
                        <option value="00">00</option>
                        <option value="01">01</option>
                        <option value="02">02</option>
                        <option value="03">03</option>
                        <option value="04">04</option>
                        <option value="05">05</option>
                        <option value="06">06</option>
                        <option value="07">07</option>
                        <option value="08">08</option>
                        <option value="09">09</option>
                        <option value="10">10</option>
                        <option value="11">11</option>
                        <option value="12">12</option>
                        <option value="13">13</option>
                        <option value="14">14</option>
                        <option value="15">15</option>
                        <option value="16">16</option>
                        <option value="17">17</option>
                        <option value="18">18</option>
                        <option value="19">19</option>
                        <option value="20">20</option>
                        <option value="21">21</option>
                        <option value="22">22</option>
                        <option value="23">23</option>
                        <option value="24">24</option>
                        <option value="25">25</option>
                        <option value="26">26</option>
                        <option value="27">27</option>
                        <option value="28">28</option>
                        <option value="29">29</option>
                        <option value="30">30</option>
                        <option value="31">31</option>
                        <option value="32">32</option>
                        <option value="33">33</option>
                        <option value="34">34</option>
                        <option value="35">35</option>
                        <option value="36">36</option>
                        <option value="37">37</option>
                        <option value="38">38</option>
                        <option value="39">39</option>
                        <option value="40">40</option>
                        <option value="41">41</option>
                        <option value="42">42</option>
                        <option value="43">43</option>
                        <option value="44">44</option>
                        <option value="45">45</option>
                        <option value="46">46</option>
                        <option value="47">47</option>
                        <option value="48">48</option>
                        <option value="49">49</option>
                        <option value="50">50</option>
                        <option value="51">51</option>
                        <option value="52">52</option>
                        <option value="53">53</option>
                        <option value="54">54</option>
                        <option value="55">55</option>
                        <option value="56">56</option>
                        <option value="57">57</option>
                        <option value="58">58</option>
                        <option value="59">59</option>
                    </select>
                </div>
                <div class="datetime-input-group">
                    <label>秒:</label>
                    <select id="modalSecond">
                        <option value="00">00</option>
                        <option value="01">01</option>
                        <option value="02">02</option>
                        <option value="03">03</option>
                        <option value="04">04</option>
                        <option value="05">05</option>
                        <option value="06">06</option>
                        <option value="07">07</option>
                        <option value="08">08</option>
                        <option value="09">09</option>
                        <option value="10">10</option>
                        <option value="11">11</option>
                        <option value="12">12</option>
                        <option value="13">13</option>
                        <option value="14">14</option>
                        <option value="15">15</option>
                        <option value="16">16</option>
                        <option value="17">17</option>
                        <option value="18">18</option>
                        <option value="19">19</option>
                        <option value="20">20</option>
                        <option value="21">21</option>
                        <option value="22">22</option>
                        <option value="23">23</option>
                        <option value="24">24</option>
                        <option value="25">25</option>
                        <option value="26">26</option>
                        <option value="27">27</option>
                        <option value="28">28</option>
                        <option value="29">29</option>
                        <option value="30">30</option>
                        <option value="31">31</option>
                        <option value="32">32</option>
                        <option value="33">33</option>
                        <option value="34">34</option>
                        <option value="35">35</option>
                        <option value="36">36</option>
                        <option value="37">37</option>
                        <option value="38">38</option>
                        <option value="39">39</option>
                        <option value="40">40</option>
                        <option value="41">41</option>
                        <option value="42">42</option>
                        <option value="43">43</option>
                        <option value="44">44</option>
                        <option value="45">45</option>
                        <option value="46">46</option>
                        <option value="47">47</option>
                        <option value="48">48</option>
                        <option value="49">49</option>
                        <option value="50">50</option>
                        <option value="51">51</option>
                        <option value="52">52</option>
                        <option value="53">53</option>
                        <option value="54">54</option>
                        <option value="55">55</option>
                        <option value="56">56</option>
                        <option value="57">57</option>
                        <option value="58">58</option>
                        <option value="59">59</option>
                    </select>
                </div>
            </div>
            <div class="modal-buttons">
                <button class="modal-btn modal-cancel" onclick="closeDateTimePicker()">取消</button>
                <button class="modal-btn modal-confirm" onclick="confirmDateTimePicker()">确定</button>
            </div>
        </div>
    </div>

    <script>
        // 页面加载完成后执行
        document.addEventListener('DOMContentLoaded', function() {
            loadAppNames();
            
            // 设置默认时间（最近24小时）
            setDefaultTimeRange();
        });

        // 加载AppName列表
        async function loadAppNames() {
            try {
                const response = await fetch('/api/v1/appnames');
                const appNames = await response.json();
                const select = document.getElementById('appName');
                
                appNames.forEach(name => {
                    const option = document.createElement('option');
                    option.value = name;
                    option.textContent = name;
                    select.appendChild(option);
                });
            } catch (error) {
                console.error('加载AppName失败:', error);
                alert('加载应用列表失败，请刷新页面重试');
            }
        }

        // 设置默认时间范围（最近24小时）
        function setDefaultTimeRange() {
            const now = new Date();
            const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
            
            const endInput = document.getElementById('endTime');
            const startInput = document.getElementById('startTime');
            
            endInput.value = formatDateTimeLocal(now);
            startInput.value = formatDateTimeLocal(yesterday);
        }
        
        // 全局变量存储当前操作的输入框ID
        let currentDateTimeInput = null;
        
        // 打开日期时间选择器
        function openDateTimePicker(inputId) {
            currentDateTimeInput = inputId;
            const modal = document.getElementById('datetimeModal');
            const input = document.getElementById(inputId);
            
            // 如果输入框已有值，解析并填充到模态框中
            if (input.value) {
                const date = new Date(input.value);
                if (!isNaN(date.getTime())) {
                    document.getElementById('modalDate').value = formatDateForInput(date);
                    document.getElementById('modalHour').value = String(date.getHours()).padStart(2, '0');
                    document.getElementById('modalMinute').value = String(date.getMinutes()).padStart(2, '0');
                    document.getElementById('modalSecond').value = String(date.getSeconds()).padStart(2, '0');
                } else {
                    // 如果解析失败，使用当前时间
                    setModalCurrentTime();
                }
            } else {
                // 如果没有值，使用当前时间
                setModalCurrentTime();
            }
            
            modal.style.display = 'block';
        }
        
        // 设置模态框为当前时间
        function setModalCurrentTime() {
            const now = new Date();
            document.getElementById('modalDate').value = formatDateForInput(now);
            document.getElementById('modalHour').value = String(now.getHours()).padStart(2, '0');
            document.getElementById('modalMinute').value = String(now.getMinutes()).padStart(2, '0');
            document.getElementById('modalSecond').value = String(now.getSeconds()).padStart(2, '0');
        }
        
        // 格式化日期为input type="date"的格式
        function formatDateForInput(date) {
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            return year + '-' + month + '-' + day;
        }
        
        // 确认日期时间选择
        function confirmDateTimePicker() {
            if (!currentDateTimeInput) return;
            
            const date = document.getElementById('modalDate').value;
            const hour = document.getElementById('modalHour').value;
            const minute = document.getElementById('modalMinute').value;
            const second = document.getElementById('modalSecond').value;
            
            if (!date) {
                alert('请选择日期');
                return;
            }
            
            const dateTimeString = date + 'T' + hour + ':' + minute + ':' + second;
            document.getElementById(currentDateTimeInput).value = dateTimeString;
            closeDateTimePicker();
        }
        
        // 关闭日期时间选择器
        function closeDateTimePicker() {
            document.getElementById('datetimeModal').style.display = 'none';
            currentDateTimeInput = null;
        }
        
        // 点击模态框外部关闭
        window.onclick = function(event) {
            const modal = document.getElementById('datetimeModal');
            if (event.target === modal) {
                closeDateTimePicker();
            }
        }

        // 格式化日期时间为精确到秒的格式
        function formatDateTimeLocal(date) {
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            const hours = String(date.getHours()).padStart(2, '0');
            const minutes = String(date.getMinutes()).padStart(2, '0');
            const seconds = String(date.getSeconds()).padStart(2, '0');
            
            return year + '-' + month + '-' + day + 'T' + hours + ':' + minutes + ':' + seconds;
        }

        // 表单提交处理
        document.getElementById('qosForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const submitBtn = document.querySelector('.submit-btn');
            const loading = document.getElementById('loading');
            const results = document.getElementById('results');
            
            // 获取表单数据
            const formData = new FormData(e.target);
            const data = {
                appName: formData.get('appName'),
                startTime: formData.get('startTime'),
                endTime: formData.get('endTime'),
                streamId: formData.get('streamId'),
                domain: formData.get('domain'),
                uid: formData.get('uid'),
                hour: formData.get('hour'),
                fuzzySearch: formData.get('fuzzySearch') === 'on'
            };
            
            // 显示加载状态
            submitBtn.disabled = true;
            loading.style.display = 'block';
            results.innerHTML = '';
            
            try {
                const response = await fetch('/api/v1/qos', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });
                
                if (!response.ok) {
                    throw new Error('请求失败: ' + response.statusText);
                }
                
                const html = await response.text();
                results.innerHTML = html;
                
            } catch (error) {
                console.error('分析失败:', error);
                results.innerHTML = '<div style="color: red;">分析失败: ' + error.message + '</div>';
            } finally {
                // 恢复按钮状态
                submitBtn.disabled = false;
                loading.style.display = 'none';
            }
        });
    </script>
</body>
</html>
`
}

// Qos 主函数，启动QOS服务器
func Qos(config *config.Config) {
	if config == nil {
		//log.Fatal("配置不能为空")
	}

	server := NewQOSServer(config)
	server.StartServer()
}

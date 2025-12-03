package miku

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
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

	// 将reports格式化为CSV并写入文件
	csvFile, err := os.Create("qos_report.csv")
	if err != nil {
		log.Printf("创建CSV文件失败: %v", err)
	} else {
		defer csvFile.Close()

		writer := csv.NewWriter(csvFile)
		defer writer.Flush()

		// 写入CSV头
		headers := []string{
			"ClientType", "Cts", "DimIp", "DimIsp", "DimCdndomain", "DimCdnip",
			"DimCoderatebps", "DimHeartType", "DimIsInBackground", "DimLine",
			"DimNetworktype", "DimPlatform", "DimStream", "DimStreamUrl", "DimVersion",
			"FieldVideoBadQuality", "InsertTs", "LogTime", "Systs", "Minute",
			"Innerreporttime", "Innerfilepath", "Day", "Hour",
		}
		if err := writer.Write(headers); err != nil {
			log.Printf("写入CSV头失败: %v", err)
		}

		// 写入数据行
		for _, report := range reports {
			record := []string{
				s.getString(report.ClientType),
				s.getInt64(report.Cts),
				s.getString(report.DimIp),
				s.getString(report.DimIsp),
				s.getString(report.DimCdndomain),
				s.getString(report.DimCdnip),
				s.getString(report.DimCoderatebps),
				s.getString(report.DimHeartType),
				s.getString(report.DimIsInBackground),
				s.getString(report.DimLine),
				s.getString(report.DimNetworktype),
				s.getString(report.DimPlatform),
				s.getString(report.DimStream),
				s.getString(report.DimStreamUrl),
				s.getString(report.DimVersion),
				s.getInt64(report.FieldVideoBadQuality),
				s.getInt64(report.InsertTs),
				s.getInt64(report.LogTime),
				s.getInt64(report.Systs),
				s.getString(report.Minute),
				s.getInt64(report.Innerreporttime),
				s.getString(report.Innerfilepath),
				s.getString(report.Day),
				s.getString(report.Hour),
			}
			if err := writer.Write(record); err != nil {
				log.Printf("写入CSV记录失败: %v", err)
			}
		}
		log.Println("CSV报告已保存到 qos_report.csv")
	}

	// 在Go代码中实现按分钟聚合（模拟第二个SQL查询的效果）
	minuteAggregated := s.aggregateByMinute(reports)

	// 按照TestHy1方式聚合数据
	cdnAggregated, clientAggregated := s.aggregateByTestHy1(reports)

	// 生成分钟聚合数据的折线图
	minuteChartHTML := s.generateMinuteChartHTML(minuteAggregated)

	// 生成聚合数据表格HTML
	tableHTML := s.generateTableHTML(minuteAggregated, cdnAggregated, clientAggregated)

	// 合并图表和表格HTML
	fullHTML := minuteChartHTML + tableHTML

	// 返回完整的HTML
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fullHTML))
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

// generateMinuteChartHTML 生成分钟聚合数据的折线图HTML
func (s *QOSServer) generateMinuteChartHTML(minuteAggregated []MinuteAggregatedData) string {
	if len(minuteAggregated) == 0 {
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "每分钟不良质量占比趋势",
			Left:  "center",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			Formatter: "{a} <br/>{b} : {c}%",
			Show:      opts.Bool(true),
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
				Formatter: "{value}%",
			},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "100%",
			Height: "400px",
			Theme:  "white",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
	)

	// 准备X轴数据（时间戳）
	var xAxisData []string
	// 准备Y轴数据（百分比）
	var yAxisData []opts.LineData

	for _, data := range minuteAggregated {
		xAxisData = append(xAxisData, data.Timestamp)
		yAxisData = append(yAxisData, opts.LineData{
			Value: data.Percent,
		})
	}
	log.Println(xAxisData, yAxisData)

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("不良质量占比", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#dc3545",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#dc3545",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 使用iframe包装图表HTML作为web component
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">每分钟不良质量占比趋势图</h2>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 450px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, encodedHTML)

	return htmlContent
}

// formatArrayToJS 将字符串数组格式化为JavaScript数组
func formatArrayToJS(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}

	var jsArray strings.Builder
	jsArray.WriteString("[")
	for i, item := range arr {
		if i > 0 {
			jsArray.WriteString(",")
		}
		jsArray.WriteString(fmt.Sprintf("'%s'", item))
	}
	jsArray.WriteString("]")

	return jsArray.String()
}

// formatFloatArrayToJS 将浮点数数组格式化为JavaScript数组
func formatFloatArrayToJS(arr []float64) string {
	if len(arr) == 0 {
		return "[]"
	}

	var jsArray strings.Builder
	jsArray.WriteString("[")
	for i, item := range arr {
		if i > 0 {
			jsArray.WriteString(",")
		}
		jsArray.WriteString(fmt.Sprintf("%.2f", item))
	}
	jsArray.WriteString("]")

	return jsArray.String()
}

// generateTableHTML 生成聚合数据表格HTML
func (s *QOSServer) generateTableHTML(minuteAggregated []MinuteAggregatedData, cdnAggregated []AggregatedData, clientAggregated []ClientAggregatedData) string {
	if len(cdnAggregated) == 0 && len(clientAggregated) == 0 {
		return ""
	}

	htmlContent := `
	<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
		<h2 style="text-align: center; color: #333; margin-bottom: 30px;">聚合数据分析</h2>
	`

	// 生成按分钟聚合的表格
	if len(minuteAggregated) > 0 {
		htmlContent += `
		<h3 style="color: #555; margin-bottom: 20px;">按分钟聚合分析</h3>
		<div style="overflow-x: auto;">
			<table style="width: 100%; border-collapse: collapse; font-size: 12px;">
				<thead>
					<tr style="background-color: #f2f2f2;">
						<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">时间</th>
						<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">不良质量占比</th>
						<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">延迟次数</th>
						<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">总次数</th>
					</tr>
				</thead>
				<tbody>
		`

		for _, data := range minuteAggregated {
			percentColor := "color: #28a745;"
			if data.Percent > 0 {
				percentColor = "color: #dc3545;"
			}

			htmlContent += fmt.Sprintf(`
					<tr>
						<td style="border: 1px solid #ddd; padding: 8px;">%s</td>
						<td style="border: 1px solid #ddd; padding: 8px; %s">%.2f%%</td>
						<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
						<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
					</tr>
			`, data.Timestamp, percentColor, data.Percent, data.LagCount, data.TotalCount)
		}

		htmlContent += `
				</tbody>
			</table>
		</div>
	`
	}

	htmlContent += `
		<div style="display: grid; grid-template-columns: 1fr 1fr; gap: 30px; margin-top: 30px;">
	`

	// 生成CDN IP表格
	if len(cdnAggregated) > 0 {
		htmlContent += `
			<div class="chart-box">
				<div class="chart-title">CDN IP质量分析统计</div>
				<div style="overflow-x: auto;">
					<table style="width: 100%; border-collapse: collapse; font-size: 12px;">
						<thead>
							<tr style="background-color: #f2f2f2;">
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">CDN IP</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">不良质量次数</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">总次数</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">不良质量占比</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">延迟次数</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">无延迟次数</th>
							</tr>
						</thead>
						<tbody>
		`

		for _, data := range cdnAggregated {
			badPercentColor := "color: #28a745;"
			if data.BadPercent > 0 {
				badPercentColor = "color: #dc3545;"
			}

			htmlContent += fmt.Sprintf(`
							<tr>
								<td style="border: 1px solid #ddd; padding: 8px;">%s</td>
								<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
								<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
								<td style="border: 1px solid #ddd; padding: 8px; %s">%.2f%%</td>
								<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
								<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
							</tr>
			`, data.CDNIP, data.BadCount, data.TotalCount, badPercentColor, data.BadPercent, data.LagCount, data.NoLagCount)
		}

		htmlContent += `
						</tbody>
					</table>
				</div>
			</div>
		`
	}

	// 生成Client IP表格
	if len(clientAggregated) > 0 {
		htmlContent += `
			<div class="chart-box">
				<div class="chart-title">Client IP质量分析统计</div>
				<div style="overflow-x: auto;">
					<table style="width: 100%; border-collapse: collapse; font-size: 12px;">
						<thead>
							<tr style="background-color: #f2f2f2;">
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">Client IP</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">不良质量次数</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">总次数</th>
								<th style="border: 1px solid #ddd; padding: 8px; text-align: left;">不良质量占比</th>
							</tr>
						</thead>
						<tbody>
		`

		for _, data := range clientAggregated {
			badPercentColor := "color: #28a745;"
			if data.BadPercent > 0 {
				badPercentColor = "color: #dc3545;"
			}

			htmlContent += fmt.Sprintf(`
							<tr>
								<td style="border: 1px solid #ddd; padding: 8px;">%s</td>
								<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
								<td style="border: 1px solid #ddd; padding: 8px;">%d</td>
								<td style="border: 1px solid #ddd; padding: 8px; %s">%.2f%%</td>
							</tr>
			`, data.ClientIP, data.BadCount, data.TotalCount, badPercentColor, data.BadPercent)
		}

		htmlContent += `
						</tbody>
					</table>
				</div>
			</div>
		`
	}

	htmlContent += `
		</div>
	</div>
	`

	return htmlContent
}

// Qos 主函数，启动QOS服务器
func Qos(config *config.Config) {
	if config == nil {
		//log.Fatal("配置不能为空")
	}

	server := NewQOSServer(config)
	server.StartServer()
}

package miku

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
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
	sql := s.buildSQLQuery(req)
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
func (s *QOSServer) buildSQLQuery(req QOSRequest) string {
	// 基础查询
	sql := "SELECT * FROM huyabiz_quality_report_log WHERE 1=1 "

	// 添加AppName过滤
	if req.AppName != "" {
		//sql += fmt.Sprintf(" AND dim_platform = '%s'", req.AppName)
	}

	// 添加时间范围过滤（转换为day格式）
	if req.StartTime != "" {
		//startDay := s.convertToDay(req.StartTime)
		//sql += fmt.Sprintf(" AND day >= '%s'", startDay)
	}

	if req.EndTime != "" {
		//endDay := s.convertToDay(req.EndTime)
		//sql += fmt.Sprintf(" AND day <= '%s'", endDay)
	}
	sql += fmt.Sprint("AND day = '20251203'")

	// 添加流ID过滤
	if req.StreamID != "" {
		if req.FuzzySearch {
			sql += fmt.Sprintf(" AND dim_stream LIKE '%%%s%%'", req.StreamID)
		} else {
			sql += fmt.Sprintf(" AND dim_stream = '%s'", req.StreamID)
		}
	}

	// 限制结果数量
	sql += " LIMIT 100"

	return sql
}

// convertToDay 将时间字符串转换为day格式 (YYYYMMDD)
func (s *QOSServer) convertToDay(timeStr string) string {
	// 尝试解析不同格式的时间字符串
	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t.Format("20060102")
		}
	}

	// 如果解析失败，直接返回原始字符串
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

	// 使用简单HTML表格而不是echarts
	html := `
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>QOS分析结果</title>
		<style>
			body { font-family: Arial, sans-serif; margin: 20px; }
			h1 { color: #333; }
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
			th { background-color: #f2f2f2; }
			.back-btn { 
				background-color: #007bff; 
				color: white; 
				padding: 10px 20px; 
				border: none; 
				border-radius: 4px; 
				cursor: pointer;
				margin-bottom: 20px;
			}
		</style>
	</head>
	<body>
		<button class="back-btn" onclick="window.history.back()">返回</button>
		<h1>QOS质量分析报告</h1>
		<p>共找到 ` + fmt.Sprintf("%d", len(reports)) + ` 条记录</p>
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
        input[type="datetime-local"], input[type="text"], select {
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
                    <input type="datetime-local" id="startTime" name="startTime" required>
                </div>
                <div class="form-group">
                    <label for="endTime">结束日期时间:</label>
                    <input type="datetime-local" id="endTime" name="endTime" required>
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

        // 格式化日期时间为datetime-local格式
        function formatDateTimeLocal(date) {
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            const hours = String(date.getHours()).padStart(2, '0');
            const minutes = String(date.getMinutes()).padStart(2, '0');
            
            return year + '-' + month + '-' + day + 'T' + hours + ':' + minutes;
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

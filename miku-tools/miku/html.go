package miku

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"mikutool/public/util"
	"sort"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// generateTableHTML 生成聚合数据表格HTML
func (s *QOSServer) generateTableHTML(cdnAggregated []AggregatedData, clientAggregated []AggregatedData) string {
	if len(cdnAggregated) == 0 && len(clientAggregated) == 0 {
		log.Println("generateTableHTML: 聚合数据为空")
		return ""
	}

	// 生成CDN IP表格
	var cdnTableHTML string
	if len(cdnAggregated) > 0 {
		cdnTableHTML = s.generateEchartsTable(cdnAggregated, "CDN IP质量分析统计", true)
	}

	// 生成Client IP表格
	var clientTableHTML string
	if len(clientAggregated) > 0 {
		clientTableHTML = s.generateEchartsTable(clientAggregated, "Client IP质量分析统计", false)
	}

	// 合并表格HTML
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">聚合数据分析</h2>
			<div style="display: flex; flex-direction: column; gap: 30px; margin-top: 30px;">
				%s
				%s
			</div>
		</div>
	`, cdnTableHTML, clientTableHTML)

	return htmlContent
}

// generateEchartsTable 使用AG Grid Community生成表格
func (s *QOSServer) generateEchartsTable(data []AggregatedData, title string, _ bool) string {
	// 限制最多显示30行数据
	maxRows := 30
	if len(data) > maxRows {
		data = data[:maxRows]
	}

	// 准备表格数据
	var tableRows []string
	for _, item := range data {
		lagRatio := 0.0
		if item.TotalCount > 0 {
			lagRatio = float64(item.LagCount) * 100.0 / float64(item.TotalCount)
		}

		// 处理RemoteIps数组，转换为逗号分隔的字符串
		remoteIpsStr := strings.Join(item.RemoteIps, ", ")

		tableRows = append(tableRows, fmt.Sprintf(`{
			"ip": "%s",
			"lagCount": %d,
			"totalCount": %d,
			"lagRatio": "%.2f%%",
            "province": "%s",
            "isp": "%s",
            "remoteIps": "%s",
            "lagRate": "%.1f%%"
		}`, item.IP, item.LagCount, item.TotalCount, lagRatio, item.Prov, item.Isp, remoteIpsStr, item.LagRate))
	}

	tableData := strings.Join(tableRows, ",\n")
	//log.Println(tableData)

	// 创建包含AG Grid的HTML页面
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>%s</title>
    <script src="https://cdn.jsdelivr.net/npm/ag-grid-community/dist/ag-grid-community.min.js"></script>
    <style>
        body {
            margin: 0;
            padding: 20px;
            font-family: Arial, sans-serif;
        }
        #grid-container {
            height: 600px;
            width: 100%%;
        }
        .ag-theme-alpine {
            height: 100%%;
            width: 100%%;
        }
    </style>
</head>
<body>
    <h2 style="text-align: center; color: #333; margin-bottom: 20px;">%s</h2>
    <div style="margin-bottom: 15px; padding: 10px; background: #f0f8ff; border: 1px solid #b3d9ff; border-radius: 4px; font-size: 12px; color: #666;">
        <strong>使用提示:</strong> 支持多选行，按 Ctrl+Shift+C 复制选中数据到剪贴板
    </div>
    <div id="grid-container">
        <div id="myGrid" class="ag-theme-alpine"></div>
    </div>

    <script>
        // 定义表格列配置
        const columnDefs = [
            {
                headerName: "IP",
                field: "ip",
                sortable: true,
                filter: true,
                width: 150,
                comparator: (valueA, valueB) => {
                    // IP地址排序比较器
                    const partsA = valueA.split('.').map(Number);
                    const partsB = valueB.split('.').map(Number);
                    
                    for (let i = 0; i < 4; i++) {
                        if (partsA[i] !== partsB[i]) {
                            return partsA[i] - partsB[i];
                        }
                    }
                    return 0;
                }
            },
            {
                headerName: "卡顿样本数",
                field: "lagCount",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => valueA - valueB
            },
            {
                headerName: "总样本数",
                field: "totalCount",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => valueA - valueB
            },
            {
                headerName: "卡顿比",
                field: "lagRatio",
                sortable: true,
                filter: true,
                width: 120,
                comparator: (valueA, valueB) => {
                    // 百分比排序比较器
                    const numA = parseFloat(valueA.replace('%%', ''));
                    const numB = parseFloat(valueB.replace('%%', ''));
                    return numA - numB;
                }
            },
            {
                headerName: "省份",
                field: "province",
                sortable: true,
                filter: true,
                width: 100
            },
            {
                headerName: "运营商",
                field: "isp",
                sortable: true,
                filter: true,
                width: 120
            },
            {
                headerName: "remote ips",
                field: "remoteIps",
                sortable: true,
                filter: true,
                width: 200
            },
            {
                headerName: "占总体卡顿比重",
                field: "lagRate",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 140,
                comparator: (valueA, valueB) => {
                    // 百分比排序比较器
                    const numA = parseFloat(valueA.replace('%%', ''));
                    const numB = parseFloat(valueB.replace('%%', ''));
                    return numA - numB;
                }
            }
        ];

        // 定义表格数据
        const rowData = [%s];

        // 创建AG Grid实例
        const gridOptions = {
            columnDefs: columnDefs,
            rowData: rowData,
            defaultColDef: {
                resizable: true,
                sortable: true,
                filter: true
            },
            enableRangeSelection: true,
            enableRangeHandle: true,
            enableFillHandle: true,
            enableCellTextSelection: true,
            pagination: false,
            rowSelection: {
                mode: 'multiRow',
                checkboxes: true,
                headerCheckbox: true,
                enableClickSelection: true,
                enableRangeSelection: true
            },
            animateRows: true,
            suppressMenuHide: true,
            onFirstDataRendered: function(params) {
                params.api.sizeColumnsToFit();
            },
            onGridReady: function(params) {
                // 添加复制功能
                params.api.addEventListener('keydown', function(event) {
                    if (event.ctrlKey && event.key === 'c' && event.shiftKey) {
                        copySelectedData(params.api);
                        event.preventDefault();
                    }
                });
            }
        };

        // 复制选中数据的功能
        function copySelectedData(api) {
            const selectedRows = api.getSelectedRows();
            if (selectedRows && selectedRows.length > 0) {
                // 将选中数据转换为制表符分隔的文本
                const headers = Object.keys(selectedRows[0]);
                const headerText = headers.join('\\t') + '\\n';
                
                const rowsText = selectedRows.map(row => {
                    return headers.map(header => {
                        const value = row[header];
                        // 处理可能的undefined或null值
                        return value !== undefined && value !== null ? String(value) : '';
                    }).join('\\t');
                }).join('\\n');
                
                const fullText = headerText + rowsText;
                
                // 复制到剪贴板
                navigator.clipboard.writeText(fullText).then(() => {
                    // 显示复制成功的提示
                    showToast('已复制 ' + selectedRows.length + ' 行数据到剪贴板');
                }).catch(err => {
                    console.error('复制失败:', err);
                    showToast('复制失败，请重试');
                });
            } else {
                showToast('请先选择要复制的数据行');
            }
        }
        
        // 显示提示信息
        function showToast(message) {
            const toast = document.createElement('div');
            toast.style.cssText = 'position: fixed; top: 20px; right: 20px; background: #4CAF50; color: white; padding: 10px 20px; border-radius: 4px; z-index: 1000; font-family: Arial, sans-serif; font-size: 14px; box-shadow: 0 2px 8px rgba(0,0,0,0.2);';
            toast.textContent = message;
            document.body.appendChild(toast);
            
            // 3秒后自动消失
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 3000);
        }

        // 等待DOM加载完成后初始化表格
        document.addEventListener('DOMContentLoaded', function() {
            const gridDiv = document.querySelector('#myGrid');
            agGrid.createGrid(gridDiv, gridOptions);
        });
    </script>
</body>
</html>`, title, title, tableData)

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(htmlContent))

	// 返回包含iframe的HTML
	return fmt.Sprintf(`
		<div class="chart-box">
			<div style="margin-bottom: 10px; font-weight: bold; color: #333;">%s</div>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 800px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, title, encodedHTML)
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
        /* 日期时间输入框样式 */
        #startTime, #endTime {
            cursor: pointer;
            width: calc(100% - 20px);
        }
        #startTime:hover, #endTime:hover {
            background-color: #f5f5f5;
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
            width: 500px;
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
        
        /* 日历样式 */
        .calendar-content {
            display: flex;
            gap: 20px;
        }
        
        .calendar-main {
            flex: 2;
        }
        
        .calendar-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }
        
        .month-nav {
            background: none;
            border: none;
            font-size: 18px;
            cursor: pointer;
            padding: 5px 10px;
            border-radius: 4px;
        }
        
        .month-nav:hover {
            background-color: #f5f5f5;
        }
        
        .calendar-table {
            width: 100%;
            border-collapse: collapse;
        }
        
        .calendar-table th, .calendar-table td {
            text-align: center;
            padding: 8px;
            border: 1px solid #eee;
        }
        
        .calendar-table th {
            background-color: #f9f9f9;
            font-weight: normal;
        }
        
        .calendar-table td {
            cursor: pointer;
        }
        
        .calendar-table td:hover {
            background-color: #e9e9e9;
        }
        
        .calendar-table td.today {
            background-color: #d9edf7;
        }
        
        .calendar-table td.selected {
            background-color: #337ab7;
            color: white;
        }
        
        /* 时间选择样式 */
        .time-selector {
            flex: 1;
        }
        
        .time-selector table {
            width: 100%;
            border-collapse: collapse;
        }
        
        .time-selector table td {
            padding: 5px;
            text-align: center;
        }
        
        .time-selector table td:last-child {
            width: 30px;
        }
        
        /* 滑块样式 */
        input[type="range"] {
            width: 100%;
        }
        
        /* 时间显示样式 */
        .time-display {
            font-size: 18px;
            font-weight: bold;
            margin-bottom: 10px;
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
                    <input type="datetime-local" step="1" id="startTime" name="startTime" required>
                </div>
                <div class="form-group">
                    <label for="endTime">结束日期时间:</label>
                    <input type="datetime-local" step="1" id="endTime" name="endTime" required>
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
                <input type="number" id="hour" name="hour" min="0" max="23">
            </div>

            <!-- 模糊搜索选项 -->
            <div class="form-group">
                <div class="checkbox-group">
                    <input type="checkbox" id="fuzzySearch" name="fuzzySearch">
                    <label for="fuzzySearch" style="margin: 0; font-weight: normal;">支持模糊查找</label>
                </div>
            </div>

            <!-- 详细日志选项 -->
            <div class="form-group">
                <div class="checkbox-group">
                    <input type="checkbox" id="detailLog" name="detailLog">
                    <label for="detailLog" style="margin: 0; font-weight: normal;">开启详细日志</label>
                </div>
            </div>

            <!-- 保留原始数据选项 -->
            <div class="form-group">
                <div class="checkbox-group">
                    <input type="checkbox" id="rawData" name="rawData">
                    <label for="rawData" style="margin: 0; font-weight: normal;">保留原始数据</label>
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
                fuzzySearch: formData.get('fuzzySearch') === 'on',
                loglevel: formData.get('detailLog') === 'on' ? 'detail' : 'normal',
                rawData: formData.get('rawData') === 'on'
            };
            console.log(data)
            
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

// generateMinuteChartHTML 生成分钟聚合数据的折线图HTML
func (s *QOSServer) generateMinuteChartHTML(minuteAggregated []MinuteAggregatedData) string {
	if len(minuteAggregated) == 0 {
		log.Println("分钟聚合数据为空")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "卡顿率",
			Left:  "right",
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
	//log.Println(xAxisData, yAxisData)

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("卡顿率", yAxisData).
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
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">卡顿率趋势图</h2>
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

// generateLagUserRatioChartHTML 生成卡顿用户占比折线图HTML
func (s *QOSServer) generateLagUserRatioChartHTML(minuteAggregated []MinuteAggregatedData) string {
	if len(minuteAggregated) == 0 {
		log.Println("分钟聚合数据为空")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "卡顿用户占比",
			Left:  "right",
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
	// 准备Y轴数据（卡顿用户占比百分比）
	var yAxisData []opts.LineData

	for _, data := range minuteAggregated {
		xAxisData = append(xAxisData, data.Timestamp)

		// 计算卡顿用户占比百分比
		var ratio float64
		if data.TotalUserCnt > 0 {
			ratio = float64(data.LagUserCnt) / float64(data.TotalUserCnt) * 100
		}

		yAxisData = append(yAxisData, opts.LineData{
			Value: ratio,
		})
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("卡顿用户占比", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#fd7e14",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#fd7e14",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成卡顿用户占比图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 使用iframe包装图表HTML作为web component
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">卡顿用户占比趋势图</h2>
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

// generateCdnLagRatioChartHTML 生成节点卡顿占比折线图HTML
func (s *QOSServer) generateCdnLagRatioChartHTML(minuteAggregated []MinuteAggregatedData) string {
	if len(minuteAggregated) == 0 {
		log.Println("分钟聚合数据为空")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "节点卡顿占比",
			Left:  "right",
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
	// 准备Y轴数据（节点卡顿占比百分比）
	var yAxisData []opts.LineData

	for _, data := range minuteAggregated {
		xAxisData = append(xAxisData, data.Timestamp)

		// 计算节点卡顿占比百分比
		var ratio float64
		if data.TotalCdnCnt > 0 {
			ratio = float64(data.LagCdnCnt) / float64(data.TotalCdnCnt) * 100
		}

		yAxisData = append(yAxisData, opts.LineData{
			Value: ratio,
		})
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("节点卡顿占比", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#6f42c1",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#6f42c1",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成节点卡顿占比图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 使用iframe包装图表HTML作为web component
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">节点卡顿占比趋势图</h2>
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

// generateStreamdVideoFpsChartHTML 生成推流/回源视频帧率折线图HTML
func (s *QOSServer) generateStreamdVideoFpsChartHTML(streamdFpsReports []util.StreamdFpsReport) string {
	if len(streamdFpsReports) == 0 {
		log.Println("视频帧率数据为空")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "推流/回源视频帧率",
			Left:  "right",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			Formatter: "{a} <br/>{b} : {c} FPS",
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
				Formatter: "{value} FPS",
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

	// 准备X轴数据（时间戳）和Y轴数据（视频帧率）
	var xAxisData []string
	var yAxisData []opts.LineData

	for _, report := range streamdFpsReports {
		if report.Ts != nil && report.Avg_IncomingVideoFps != nil {
			xAxisData = append(xAxisData, *report.Ts)
			yAxisData = append(yAxisData, opts.LineData{
				Value: *report.Avg_IncomingVideoFps,
			})
		}
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("视频帧率", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#28a745",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#28a745",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成视频帧率图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 使用iframe包装图表HTML作为web component
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">推流/回源视频帧率趋势图</h2>
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

// generateStreamdAudioFpsChartHTML 生成推流/回源音频帧率折线图HTML
func (s *QOSServer) generateStreamdAudioFpsChartHTML(streamdFpsReports []util.StreamdFpsReport) string {
	if len(streamdFpsReports) == 0 {
		log.Println("音频帧率数据为空")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "推流/回源音频帧率",
			Left:  "right",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			Formatter: "{a} <br/>{b} : {c} FPS",
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
				Formatter: "{value} FPS",
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

	// 准备X轴数据（时间戳）和Y轴数据（音频帧率）
	var xAxisData []string
	var yAxisData []opts.LineData

	for _, report := range streamdFpsReports {
		if report.Ts != nil && report.Avg_IncomingAudioFps != nil {
			xAxisData = append(xAxisData, *report.Ts)
			yAxisData = append(yAxisData, opts.LineData{
				Value: *report.Avg_IncomingAudioFps,
			})
		}
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("音频帧率", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#ffc107",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#ffc107",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成音频帧率图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 使用iframe包装图表HTML作为web component
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">推流/回源音频帧率趋势图</h2>
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

// generateStreamCntChartHTML 生成在线流个数折线图HTML
func (s *QOSServer) generateStreamCntChartHTML(streamCntReports []util.StreamdStreamCntReport) string {
	if len(streamCntReports) == 0 {
		log.Println("在线流个数数据为空")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "在线流个数",
			Left:  "right",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			Formatter: "{a} <br/>{b} : {c}",
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
				Formatter: "{value}",
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

	// 准备X轴数据（时间戳）和Y轴数据（在线流个数）
	var xAxisData []string
	var yAxisData []opts.LineData

	for _, report := range streamCntReports {
		if report.Ts_m != nil && report.StreamCnt != nil {
			xAxisData = append(xAxisData, *report.Ts_m)
			yAxisData = append(yAxisData, opts.LineData{
				Value: *report.StreamCnt,
			})
		}
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("在线流个数", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#20c997",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#20c997",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成在线流个数图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 使用iframe包装图表HTML作为web component
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">在线流个数趋势图</h2>
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

// generateUpstreamBandwidthChartHTML 生成推流/回源带宽折线图HTML
func (s *QOSServer) generateUpstreamBandwidthChartHTML(streamUpstreamBandwidthReports []util.StreamdUpstreamBandWidthReport) string {
	if len(streamUpstreamBandwidthReports) == 0 {
		log.Println("推流/回源带宽数据为空")
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "推流/回源带宽",
			Left:  "right",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			Formatter: "{a} <br/>{b} : {c} Mbps",
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
				Formatter: "{value} Mbps",
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

	// 准备X轴数据（时间戳）和Y轴数据（带宽）
	var xAxisData []string
	var yAxisData []opts.LineData

	for _, report := range streamUpstreamBandwidthReports {
		if report.Ts != nil && report.BandWidth != nil {
			xAxisData = append(xAxisData, *report.Ts)
			yAxisData = append(yAxisData, opts.LineData{
				Value: *report.BandWidth,
			})
		}
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries("带宽", yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#6610f2",
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   "#6610f2",
				Opacity: opts.Float(0.1),
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := line.Render(&buf)
	if err != nil {
		log.Println("生成推流/回源带宽图表HTML失败:", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	// 使用iframe包装图表HTML作为web component
	htmlContent := fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">推流/回源带宽趋势图</h2>
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

// generateLineChartHTML 生成简单的折线图HTML
func (s *QOSServer) generateLineChartHTML(reports []util.StreamdLagReport, xField, yField, title string) string {
	if len(reports) == 0 {
		return ""
	}

	// 创建折线图
	line := charts.NewLine()

	// 设置全局选项
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
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
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "100%",
			Height: "250px",
			Theme:  "white",
		}),
	)

	// 准备X轴和Y轴数据
	var xAxisData []string
	var yAxisData []opts.LineData

	for _, report := range reports {
		// 根据字段名获取对应的值
		var xValue, yValue string

		// 获取X轴数据
		if xField == "Ts_m" && report.Ts_m != nil {
			xValue = *report.Ts_m
		}

		// 获取Y轴数据
		if yField == "用户百秒卡顿率" && report.Ratio_lag_player != nil {
			yValue = fmt.Sprintf("%.2f", *report.Ratio_lag_player)
		} else if yField == "内部回源百秒卡顿率" && report.Ratio_lag_internal_player != nil {
			yValue = fmt.Sprintf("%.2f", *report.Ratio_lag_internal_player)
		} else if yField == "内部回源重试率" && report.Retry_ratio_puller != nil {
			yValue = fmt.Sprintf("%.2f", *report.Retry_ratio_puller)
		} else if yField == "内部回源重试次数" && report.TotalRetryTimes != nil {
			yValue = fmt.Sprintf("%d", *report.TotalRetryTimes)
		} else if yField == "回客户源站百秒卡顿率" && report.Ratio_lag_puller != nil {
			yValue = fmt.Sprintf("%.2f", *report.Ratio_lag_puller)
		}

		if xValue != "" && yValue != "" {
			xAxisData = append(xAxisData, xValue)
			yAxisData = append(yAxisData, opts.LineData{
				Value: yValue,
			})
		}
	}

	if len(xAxisData) == 0 {
		return ""
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries(title, yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(false),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#1890ff",
				Width: 2,
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

	// 返回data URL
	return fmt.Sprintf("data:text/html;base64,%s", encodedHTML)
}

func (s *QOSServer) generateStreamdChartsHTML(streamdReports []util.StreamdLagReport) string {
	// 生成streamdReports的5个折线图
	streamdChartsHTML := fmt.Sprintf(`
	<div style="margin-top: 20px;">
		<h3 style="text-align: center; color: #333; margin-bottom: 20px;">流媒体延迟分析图表</h3>
		<div style="margin-bottom: 30px;">
			<h4>用户百秒卡顿率</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
		<div style="margin-bottom: 30px;">
			<h4>内部回源百秒卡顿率</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
		<div style="margin-bottom: 30px;">
			<h4>内部回源重试率</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
		<div style="margin-bottom: 30px;">
			<h4>内部回源重试次数</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
		<div style="margin-bottom: 30px;">
			<h4>回客户源站百秒卡顿率</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
	</div>`,
		s.generateLineChartHTML(streamdReports, "Ts_m", "用户百秒卡顿率", "用户百秒卡顿率"),
		s.generateLineChartHTML(streamdReports, "Ts_m", "内部回源百秒卡顿率", "内部回源百秒卡顿率"),
		s.generateLineChartHTML(streamdReports, "Ts_m", "内部回源重试率", "内部回源重试率"),
		s.generateLineChartHTML(streamdReports, "Ts_m", "内部回源重试次数", "内部回源重试次数"),
		s.generateLineChartHTML(streamdReports, "Ts_m", "回客户源站百秒卡顿率", "回客户源站百秒卡顿率"),
	)
	return streamdChartsHTML
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

// generateAggDataChartsHTML 生成聚合数据的三个饼图
func (s *QOSServer) generateAggDataChartsHTML(aggData AggData) string {
	var chartsHTML string

	// 生成国家分布饼图
	if len(aggData.CountryCntMap) > 0 {
		chartsHTML += s.generatePieChartHTML(aggData.CountryCntMap, "国家")
	}

	// 生成区域分布饼图
	if len(aggData.AreaCntMap) > 0 {
		chartsHTML += s.generatePieChartHTML(aggData.AreaCntMap, "区域")
	}

	// 生成省份分布饼图
	if len(aggData.ProvCntMap) > 0 {
		chartsHTML += s.generatePieChartHTML(aggData.ProvCntMap, "省份")
	}

	if len(aggData.AreaLagCntMap) > 0 {
		chartsHTML += s.generatePieChartHTML(aggData.AreaLagCntMap, "区域延迟分布")
	}

	if len(aggData.ProvLagCntMap) > 0 {
		chartsHTML += s.generatePieChartHTML(aggData.ProvLagCntMap, "省份延迟分布")
	}

	return chartsHTML
}

// generatePieChartHTML 生成饼图HTML
func (s *QOSServer) generatePieChartHTML(data map[string]int, title string) string {
	if len(data) == 0 {
		return ""
	}

	// 排序数据（按值降序）
	type DataItem struct {
		Name  string
		Value int
	}

	var sortedData []DataItem
	for name, value := range data {
		sortedData = append(sortedData, DataItem{Name: name, Value: value})
	}

	sort.Slice(sortedData, func(i, j int) bool {
		return sortedData[i].Value > sortedData[j].Value
	})

	// 限制最多显示10个
	if len(sortedData) > 10 {
		sortedData = sortedData[:10]
	}

	// 创建饼图
	pie := charts.NewPie()

	// 设置全局选项
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
			Left:  "center",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "item",
			Formatter: "{a} <br/>{b}: {c} ({d}%)",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true),
			Bottom: "0%",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "100%",
			Height: "400px",
			Theme:  "white",
		}),
	)

	// 准备数据
	var pieData []opts.PieData
	for _, item := range sortedData {
		if item.Name == "" {
			item.Name = "未知"
		}
		pieData = append(pieData, opts.PieData{
			Name:  item.Name,
			Value: item.Value,
		})
	}

	// 添加数据系列
	pie.AddSeries(title, pieData).
		SetSeriesOptions(
			charts.WithPieChartOpts(opts.PieChart{
				Radius:   "60%",
				RoseType: "radius",
			}),
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{b}: {c} ({d}%)",
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := pie.Render(&buf)
	if err != nil {
		log.Printf("生成饼图HTML失败: %v", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	return fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">%s分布图</h2>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 450px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, title, encodedHTML)
}

// generateUpstreamDistributeChartsHTML 生成源站分布饼图HTML
func (s *QOSServer) generateUpstreamDistributeChartsHTML(areaCntMap, provCntMap map[string]int) string {
	var chartsHTML string

	// 生成大区分布饼图
	if len(areaCntMap) > 0 {
		chartsHTML += s.generateUpstreamPieChartHTML(areaCntMap, "源站按大区分布")
	}

	// 生成省份分布饼图
	if len(provCntMap) > 0 {
		chartsHTML += s.generateUpstreamPieChartHTML(provCntMap, "源站按省分布")
	}

	return chartsHTML
}

// generateUpstreamPieChartHTML 生成源站分布饼图HTML
func (s *QOSServer) generateUpstreamPieChartHTML(data map[string]int, title string) string {
	if len(data) == 0 {
		return ""
	}

	// 排序数据（按值降序）
	type DataItem struct {
		Name  string
		Value int
	}

	var sortedData []DataItem
	for name, value := range data {
		sortedData = append(sortedData, DataItem{Name: name, Value: value})
	}

	sort.Slice(sortedData, func(i, j int) bool {
		return sortedData[i].Value > sortedData[j].Value
	})

	// 限制最多显示10个
	if len(sortedData) > 10 {
		sortedData = sortedData[:10]
	}

	// 创建饼图
	pie := charts.NewPie()

	// 设置全局选项
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
			Left:  "center",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "item",
			Formatter: "{a} <br/>{b}: {c} ({d}%)",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true),
			Bottom: "0%",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "100%",
			Height: "400px",
			Theme:  "white",
		}),
	)

	// 准备数据
	var pieData []opts.PieData
	for _, item := range sortedData {
		if item.Name == "" {
			item.Name = "未知"
		}
		pieData = append(pieData, opts.PieData{
			Name:  item.Name,
			Value: item.Value,
		})
	}

	// 添加数据系列
	pie.AddSeries(title, pieData).
		SetSeriesOptions(
			charts.WithPieChartOpts(opts.PieChart{
				Radius:   "60%",
				RoseType: "radius",
			}),
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{b}: {c} ({d}%)",
			}),
		)

	// 生成完整的图表HTML页面
	var buf bytes.Buffer
	err := pie.Render(&buf)
	if err != nil {
		log.Printf("生成源站分布饼图HTML失败: %v", err)
		return ""
	}
	chartHTML := buf.String()

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(chartHTML))

	return fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">%s</h2>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 450px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, title, encodedHTML)
}

// generateCDNLagTableHTML 生成CDN卡顿用户表格
func (s *QOSServer) generateCDNLagTableHTML(cdnAggDatas []CdnAggregateData) string {
	if len(cdnAggDatas) == 0 {
		return ""
	}

	// 准备表格数据
	var tableRows []string
	for _, data := range cdnAggDatas {
		lagClientCount := len(data.LagIps)
		clientList := strings.Join(data.LagIps, ", ")
		tableRows = append(tableRows, fmt.Sprintf(`{
			"cdnip": "%s",
			"count": %d,
			"total": %d,
			"percent": %.2f,
			"clients": "%s"
		}`, data.IP, lagClientCount, data.TotalUserCount, float32(len(data.LagIps)*100)/float32(data.TotalUserCount), clientList))
	}

	tableData := strings.Join(tableRows, ",\n")

	// 创建包含AG Grid的HTML页面
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>CDN卡顿用户统计</title>
    <script src="https://cdn.jsdelivr.net/npm/ag-grid-community/dist/ag-grid-community.min.js"></script>
    <style>
        body {
            margin: 0;
            padding: 20px;
            font-family: Arial, sans-serif;
        }
        #grid-container {
            height: 500px;
            width: 100%%;
        }
        .ag-theme-alpine {
            height: 100%%;
            width: 100%%;
        }
    </style>
</head>
<body>
    <h2 style="text-align: center; color: #333; margin-bottom: 20px;">CDN卡顿用户统计</h2>
    <div style="margin-bottom: 15px; padding: 10px; background: #f0f8ff; border: 1px solid #b3d9ff; border-radius: 4px; font-size: 12px; color: #666;">
        <strong>使用提示:</strong> 支持多选行，按 Ctrl+Shift+C 复制选中数据到剪贴板
    </div>
    <div id="grid-container">
        <div id="myGrid" class="ag-theme-alpine"></div>
    </div>

    <script>
        // 定义表格列配置
        const columnDefs = [
            {
                headerName: "CDN IP",
                field: "cdnip",
                sortable: true,
                filter: true,
                width: 180,
                comparator: (valueA, valueB) => {
                    // IP地址排序比较器
                    const partsA = valueA.split('.').map(Number);
                    const partsB = valueB.split('.').map(Number);
                    
                    for (let i = 0; i < 4; i++) {
                        if (partsA[i] !== partsB[i]) {
                            return partsA[i] - partsB[i];
                        }
                    }
                    return 0;
                }
            },
            {
                headerName: "卡顿用户个数",
                field: "count",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => valueA - valueB,
                cellStyle: function(params) {
                    // 根据用户数量设置不同的颜色
                    if (params.value >= 10) {
                        return { color: 'red', fontWeight: 'bold' };
                    } else if (params.value >= 5) {
                        return { color: 'orange', fontWeight: 'bold' };
                    } else {
                        return { color: 'green' };
                    }
                }
            },
            {
                headerName: "总用户数",
                field: "total",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => valueA - valueB
            },
            {
                headerName: "卡顿比例",
                field: "percent",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => valueA - valueB,
                cellRenderer: function(params) {
                    return params.value + '%%';
                },
                cellStyle: function(params) {
                    // 根据卡顿比例设置不同的颜色
                    if (params.value >= 50) {
                        return { color: 'red', fontWeight: 'bold' };
                    } else if (params.value >= 20) {
                        return { color: 'orange', fontWeight: 'bold' };
                    } else {
                        return { color: 'green' };
                    }
                }
            },
            {
                headerName: "卡顿用户IP列表",
                field: "clients",
                sortable: true,
                filter: true,
                flex: 1,
                cellRenderer: function(params) {
                    // 将长列表进行格式化显示
                    const clients = params.value;
                    if (clients.length > 100) {
                        return '<span title="' + clients + '">' + clients.substring(0, 100) + '...</span>';
                    }
                    return '<span title="' + clients + '">' + clients + '</span>';
                }
            }
        ];

        // 定义表格数据
        const rowData = [%s];

        // 创建AG Grid实例
        const gridOptions = {
            columnDefs: columnDefs,
            rowData: rowData,
            defaultColDef: {
                resizable: true,
                sortable: true,
                filter: true
            },
            enableRangeSelection: true,
            enableRangeHandle: true,
            enableFillHandle: true,
            enableCellTextSelection: true,
            pagination: false,
            rowSelection: {
                mode: 'multiRow',
                checkboxes: true,
                headerCheckbox: true,
                enableClickSelection: true,
                enableRangeSelection: true
            },
            animateRows: true,
            suppressMenuHide: true,
            onFirstDataRendered: function(params) {
                params.api.sizeColumnsToFit();
            },
            onGridReady: function(params) {
                // 添加复制功能
                params.api.addEventListener('keydown', function(event) {
                    if (event.ctrlKey && event.key === 'c' && event.shiftKey) {
                        copySelectedData(params.api);
                        event.preventDefault();
                    }
                });
            }
        };

        // 复制选中数据的功能
        function copySelectedData(api) {
            const selectedRows = api.getSelectedRows();
            if (selectedRows && selectedRows.length > 0) {
                // 将选中数据转换为制表符分隔的文本
                const headers = Object.keys(selectedRows[0]);
                const headerText = headers.join('\\t') + '\\n';
                
                const rowsText = selectedRows.map(row => {
                    return headers.map(header => {
                        const value = row[header];
                        // 处理可能的undefined或null值
                        return value !== undefined && value !== null ? String(value) : '';
                    }).join('\\t');
                }).join('\\n');
                
                const fullText = headerText + rowsText;
                
                // 复制到剪贴板
                navigator.clipboard.writeText(fullText).then(() => {
                    // 显示复制成功的提示
                    showToast('已复制 ' + selectedRows.length + ' 行数据到剪贴板');
                }).catch(err => {
                    console.error('复制失败:', err);
                    showToast('复制失败，请重试');
                });
            } else {
                showToast('请先选择要复制的数据行');
            }
        }
        
        // 显示提示信息
        function showToast(message) {
            const toast = document.createElement('div');
            toast.style.cssText = 'position: fixed; top: 20px; right: 20px; background: #4CAF50; color: white; padding: 10px 20px; border-radius: 4px; z-index: 1000; font-family: Arial, sans-serif; font-size: 14px; box-shadow: 0 2px 8px rgba(0,0,0,0.2);';
            toast.textContent = message;
            document.body.appendChild(toast);
            
            // 3秒后自动消失
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 3000);
        }

        // 等待DOM加载完成后初始化表格
        document.addEventListener('DOMContentLoaded', function() {
            const gridDiv = document.querySelector('#myGrid');
            agGrid.createGrid(gridDiv, gridOptions);
        });
    </script>
</body>
</html>`, tableData)

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(htmlContent))

	// 返回包含iframe的HTML
	return fmt.Sprintf(`
		<div class="chart-box">
			<div style="margin-bottom: 10px; font-weight: bold; color: #333;">CDN卡顿用户统计</div>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 700px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, encodedHTML)
}

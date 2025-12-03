package miku

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// generateTableHTML 生成聚合数据表格HTML
func (s *QOSServer) generateTableHTML(cdnAggregated []AggregatedData, clientAggregated []ClientAggregatedData) string {
	if len(cdnAggregated) == 0 && len(clientAggregated) == 0 {
		log.Println("generateTableHTML: 聚合数据为空")
		return ""
	}

	htmlContent := `
	<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
		<h2 style="text-align: center; color: #333; margin-bottom: 30px;">聚合数据分析</h2>
	`

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

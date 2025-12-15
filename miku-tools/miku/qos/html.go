package qos

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
func GenerateTableHTML(cdnAggregated, clientAggregated []AggregatedData) string {
	if len(cdnAggregated) == 0 && len(clientAggregated) == 0 {
		log.Println("generateTableHTML: 聚合数据为空")
		return ""
	}

	// 生成CDN IP表格
	var cdnTableHTML string
	if len(cdnAggregated) > 0 {
		cdnTableHTML = generateEchartsTable(cdnAggregated, "CDN IP质量分析统计", true)
	}

	// 生成Client IP表格
	var clientTableHTML string
	if len(clientAggregated) > 0 {
		clientTableHTML = generateEchartsTable(clientAggregated, "Client IP质量分析统计", false)
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
func generateEchartsTable(data []AggregatedData, title string, _ bool) string {
	// 限制最多显示30行数据
	/*
		maxRows := 30
		if len(data) > maxRows {
			data = data[:maxRows]
		}
	*/

	// 准备表格数据
	var tableRows []string
	for _, item := range data {
		lagRatio := 0.0
		if item.TotalCount > 0 {
			lagRatio = float64(item.LagCount) * 100.0 / float64(item.TotalCount)
		}

		// 处理RemoteIps数组，转换为逗号分隔的字符串
		remoteIpsStr := strings.Join(item.NormalIps, ", ")

		// 处理子表格数据
		normalIpsJson := strings.ReplaceAll(strings.Join(item.NormalIps, `","`), `"`, `\"`)
		lagIpsJson := strings.ReplaceAll(strings.Join(item.LagIps, `","`), `"`, `\"`)
		normalStreamsJson := strings.ReplaceAll(strings.Join(item.NormalStreams, `","`), `"`, `\"`)
		lagStreamsJson := strings.ReplaceAll(strings.Join(item.LagStreams, `","`), `"`, `\"`)

		tableRows = append(tableRows, fmt.Sprintf(`{
			"ip": "%s",
			"lagCount": %d,
			"totalCount": %d,
			"lagRatio": "%.2f%%",
            "province": "%s",
            "isp": "%s",
            "remoteIps": "%s",
            "lagRate": "%.1f%%",
            "lagUsrCnt": %d,
            "totalUsrCnt": %d,
            "lagUsrRate": "%.1f%%",
            "normalIps": ["%s"],
            "lagIps": ["%s"],
            "normalStreams": ["%s"],
            "lagStreams": ["%s"]
		}`, item.IP, item.LagCount, item.TotalCount, lagRatio, item.Prov, item.Isp, remoteIpsStr, item.LagRate,
			item.LagUsrCnt, item.TotalUsrCnt, item.LagUsrRate,
			normalIpsJson, lagIpsJson, normalStreamsJson, lagStreamsJson))
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
                headerName: "",
                field: "expand",
                width: 50,
                suppressSizeToFit: true,
                sortable: false,
                filter: false,
                resizable: false,
                cellRenderer: function(params) {
                    const button = document.createElement('button');
                    button.innerHTML = '▶';
                    button.style.border = 'none';
                    button.style.background = 'none';
                    button.style.cursor = 'pointer';
                    button.style.fontSize = '12px';
                    button.style.padding = '2px';
                    button.title = '点击展开详细信息';
                    button.onclick = function(event) {
                        event.stopPropagation();
                        console.log('Button clicked for row:', params.node.rowIndex);
                        toggleDetailRow(params.node, params.api);
                    };
                    return button;
                }
            },
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
                headerName: "卡顿用户数",
                field: "lagUsrCnt",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => valueA - valueB
            },
            {
                headerName: "总用户数",
                field: "totalUsrCnt",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => valueA - valueB
            },
            {
                headerName: "卡顿用户比",
                field: "lagUsrRate",
                sortable: true,
                filter: 'agNumberColumnFilter',
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

        // 创建主表格实例
        const gridOptions = {
            columnDefs: columnDefs,
            rowData: rowData,
            defaultColDef: {
                resizable: true,
                sortable: true,
                filter: true
            },
            enableRangeHandle: true,
            enableFillHandle: true,
            enableCellTextSelection: true,
            pagination: false,
            rowSelection: {
                mode: 'multiRow',
                checkboxes: true,
                headerCheckbox: true,
                enableClickSelection: true,
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
        
        // 展开/折叠详情行
        function toggleDetailRow(node, api) {
            // 使用行索引作为唯一标识符
            const rowIndex = node.rowIndex;
            const nodeId = 'row-' + rowIndex;
            const existingDetail = document.getElementById('detail-row-' + nodeId);
            
            console.log('Toggle detail row:', nodeId, 'existing:', !!existingDetail);
            
            if (existingDetail) {
                // 折叠
                existingDetail.remove();
                node.expanded = false;
                // 更新展开按钮
                updateExpandButton(node, false);
            } else {
                // 展开
                createDetailRow(node, api, nodeId);
                node.expanded = true;
                // 更新展开按钮
                updateExpandButton(node, true);
            }
        }
        
        // 更新展开按钮状态
        function updateExpandButton(node, expanded) {
            // 使用AG Grid API获取行的DOM元素
            const rowElement = node.rowElement;
            console.log('Updating button for row:', node.rowIndex, 'expanded:', expanded, 'rowElement:', !!rowElement);
            
            if (rowElement) {
                // 查找展开列中的按钮
                const expandCell = rowElement.querySelector('[col-id="expand"]');
                console.log('Expand cell found:', !!expandCell);
                
                if (expandCell) {
                    const button = expandCell.querySelector('button');
                    console.log('Button found:', !!button);
                    
                    if (button) {
                        button.innerHTML = expanded ? '▼' : '▶';
                        console.log('Button updated to:', expanded ? '▼' : '▶');
                    }
                }
            }
        }

        // 创建详情行
        function createDetailRow(node, api, nodeId) {
            const data = node.data;
            
            console.log('Creating detail row for:', nodeId, data);
            
            // 创建详情行容器
            const detailRow = document.createElement('div');
            detailRow.id = 'detail-row-' + nodeId;
            detailRow.style.cssText = 'background: #f8f9fa; border: 1px solid #dee2e6; margin: 5px 0; padding: 15px; border-radius: 4px;';
            
            // 创建子表格容器布局
            const gridsContainer = document.createElement('div');
            gridsContainer.style.cssText = 'display: grid; grid-template-columns: 1fr 1fr; gap: 20px;';
            
            // 检查数据并创建子表格
            if (data.normalIps && data.normalIps.length > 0) {
                const normalIpsGrid = createSubGrid('正常IP列表', data.normalIps, 'normalIps');
                gridsContainer.appendChild(normalIpsGrid);
            }
            
            if (data.lagIps && data.lagIps.length > 0) {
                const lagIpsGrid = createSubGrid('卡顿IP列表', data.lagIps, 'lagIps');
                gridsContainer.appendChild(lagIpsGrid);
            }
            
            if (data.normalStreams && data.normalStreams.length > 0) {
                const normalStreamsGrid = createSubGrid('正常流列表', data.normalStreams, 'normalStreams');
                gridsContainer.appendChild(normalStreamsGrid);
            }
            
            if (data.lagStreams && data.lagStreams.length > 0) {
                const lagStreamsGrid = createSubGrid('卡顿流列表', data.lagStreams, 'lagStreams');
                gridsContainer.appendChild(lagStreamsGrid);
            }
            
            detailRow.appendChild(gridsContainer);
            
            // 将详情行插入到当前行之后
            const rowElement = node.rowElement;
            if (rowElement && rowElement.parentNode) {
                rowElement.parentNode.insertBefore(detailRow, rowElement.nextSibling);
            }
        }
        
        // 创建子表格
        function createSubGrid(title, data, type) {
            const container = document.createElement('div');
            container.style.cssText = 'border: 1px solid #ddd; border-radius: 4px; overflow: hidden;';
            
            // 创建标题
            const header = document.createElement('div');
            header.style.cssText = 'background: #e9ecef; padding: 8px 12px; font-weight: bold; color: #495057; border-bottom: 1px solid #ddd;';
            header.textContent = title + ' (' + data.length + ')';
            container.appendChild(header);
            
            // 创建表格容器
            const gridContainer = document.createElement('div');
            gridContainer.style.cssText = 'height: 200px;';
            gridContainer.id = type + '-grid-' + Math.random().toString(36).substr(2, 9);
            container.appendChild(gridContainer);
            
            // 定义列配置
            let columnDefs;
            if (type.includes('Ips')) {
                columnDefs = [
                    { headerName: "IP地址", field: "ip", sortable: true, filter: true, width: 150 }
                ];
                // 将IP数组转换为对象数组
                const rowData = data.map(ip => ({ ip: ip }));
                
                // 创建AG Grid实例
                const gridOptions = {
                    columnDefs: columnDefs,
                    rowData: rowData,
                    defaultColDef: {
                        resizable: true,
                        sortable: true,
                        filter: true
                    },
                    headerHeight: 30,
                    rowHeight: 25
                };
                
                // 延迟创建网格，确保DOM已渲染
                setTimeout(() => {
                    try {
                        // 使用新的AG Grid API
                        if (agGrid.Grid) {
                            new agGrid.Grid(gridContainer, gridOptions);
                        } else {
                            agGrid.createGrid(gridContainer, gridOptions);
                        }
                    } catch(error) {
                        console.error('Error creating sub grid:', error);
                    }
                }, 100);
                
            } else if (type.includes('Streams')) {
                columnDefs = [
                    { headerName: "流ID", field: "stream", sortable: true, filter: true, width: 300 }
                ];
                // 将流数组转换为对象数组
                const rowData = data.map(stream => ({ stream: stream }));
                
                // 创建AG Grid实例
                const gridOptions = {
                    columnDefs: columnDefs,
                    rowData: rowData,
                    defaultColDef: {
                        resizable: true,
                        sortable: true,
                        filter: true
                    },
                    headerHeight: 30,
                    rowHeight: 25
                };
                
                // 延迟创建网格，确保DOM已渲染
                setTimeout(() => {
                    try {
                        // 使用新的AG Grid API
                        if (agGrid.Grid) {
                            new agGrid.Grid(gridContainer, gridOptions);
                        } else {
                            agGrid.createGrid(gridContainer, gridOptions);
                        }
                    } catch(error) {
                        console.error('Error creating sub grid:', error);
                    }
                }, 100);
            }
            
            return container;
        }

        // 等待DOM加载完成后初始化表格
        document.addEventListener('DOMContentLoaded', function() {
            try {
                const gridDiv = document.querySelector('#myGrid');
                // 使用gridOptionsapi方式
                gridOptions.api = {};
                agGrid.createGrid(gridDiv, gridOptions);
                console.log('Main grid created successfully');
            } catch(error) {
                console.error('Error creating main grid:', error);
                // 备用方案：手动创建表格
                console.log('Trying fallback table creation');
                createFallbackTable();
            }
        });
        
        // 备用表格创建函数
        function createFallbackTable() {
            const gridDiv = document.querySelector('#myGrid');
            const table = document.createElement('table');
            table.style.cssText = 'width: 100%; border-collapse: collapse;';
            
            // 创建表头
            const thead = document.createElement('thead');
            const headerRow = document.createElement('tr');
            headerRow.innerHTML = '<th style="border: 1px solid #ddd; padding: 8px; background: #f5f5f5;">展开</th><th style="border: 1px solid #ddd; padding: 8px; background: #f5f5f5;">IP</th><th style="border: 1px solid #ddd; padding: 8px; background: #f5f5f5;">卡顿样本数</th><th style="border: 1px solid #ddd; padding: 8px; background: #f5f5f5;">总样本数</th>';
            thead.appendChild(headerRow);
            table.appendChild(thead);
            
            // 创建表体
            const tbody = document.createElement('tbody');
            rowData.forEach((row, index) => {
                const tr = document.createElement('tr');
                tr.innerHTML = '<td style="border: 1px solid #ddd; padding: 8px;">' +
                    '<button onclick="toggleFallbackDetail(' + index + ')" style="background: none; border: none; cursor: pointer;">▶</button>' +
                    '</td>' +
                    '<td style="border: 1px solid #ddd; padding: 8px;">' + row.ip + '</td>' +
                    '<td style="border: 1px solid #ddd; padding: 8px;">' + row.lagCount + '</td>' +
                    '<td style="border: 1px solid #ddd; padding: 8px;">' + row.totalCount + '</td>';
                tbody.appendChild(tr);
            });
            table.appendChild(tbody);
            
            gridDiv.innerHTML = '';
            gridDiv.appendChild(table);
        }
    </script>
</body>
</html>`, title, title, tableData)

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(htmlContent))

	// 返回包含iframe的HTML
	return fmt.Sprintf(`
		<div class="chart-box">
			<script>console.log('chart-box test 123456');</script>
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

// generateMinuteChartHTML 卡顿率趋势图
func GenerateMinuteLagRateChartHTML(minuteAggregated []util.HyLagReport) string {
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
		xAxisData = append(xAxisData, *data.Ts_m)
		yAxisData = append(yAxisData, opts.LineData{
			Value: *data.Percent,
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
func GenerateLagUserRatioChartHTML(hyLagReports []util.HyLagReport) string {
	if len(hyLagReports) == 0 {
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

	for _, data := range hyLagReports {
		xAxisData = append(xAxisData, *data.Ts_m)
		yAxisData = append(yAxisData, opts.LineData{
			Value: data.LagUsrRate,
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
func GenerateCdnLagRatioChartHTML(reports []util.HyLagReport) string {
	if len(reports) == 0 {
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

	for _, data := range reports {
		xAxisData = append(xAxisData, *data.Ts_m)
		yAxisData = append(yAxisData, opts.LineData{
			Value: *data.LagNodeRate,
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
func generateStreamdVideoFpsChartHTML(streamdFpsReports []util.StreamdFpsReport) string {
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
func generateStreamdAudioFpsChartHTML(streamdFpsReports []util.StreamdFpsReport) string {
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
func generateStreamCntChartHTML(streamCntReports []util.StreamdStreamCntReport) string {
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
func generateUpstreamBandwidthChartHTML(streamUpstreamBandwidthReports []util.StreamdUpstreamBandWidthReport) string {
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
func generateLineChartHTML(reports []util.StreamdLagReport, xField, yField, title string) string {
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

func GenerateStreamdChartsHTML(streamdReports []util.StreamdLagReport) string {
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
		generateLineChartHTML(streamdReports, "Ts_m", "用户百秒卡顿率", "用户百秒卡顿率"),
		generateLineChartHTML(streamdReports, "Ts_m", "内部回源百秒卡顿率", "内部回源百秒卡顿率"),
		generateLineChartHTML(streamdReports, "Ts_m", "内部回源重试率", "内部回源重试率"),
		generateLineChartHTML(streamdReports, "Ts_m", "内部回源重试次数", "内部回源重试次数"),
		generateLineChartHTML(streamdReports, "Ts_m", "回客户源站百秒卡顿率", "回客户源站百秒卡顿率"),
	)
	return streamdChartsHTML
}

// generateOnlineUsersChartHTML 生成在线用户数折线图HTML
func GenerateOnlineUsersChartHTML(reports []util.HyLagReport) string {
	if len(reports) == 0 {
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

	for _, data := range reports {
		xAxisData = append(xAxisData, *data.Ts_m)
		yAxisData = append(yAxisData, opts.LineData{
			Value: data.TotalUsrCnt,
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
func GenerateAggDataChartsHTML(aggData AggData) string {
	var chartsHTML string

	// 生成国家分布饼图
	if len(aggData.CountryCntMap) > 0 {
		chartsHTML += generatePieChartHTML(aggData.CountryCntMap, "国家")
	}

	// 生成区域分布饼图
	if len(aggData.AreaCntMap) > 0 {
		chartsHTML += generatePieChartHTML(aggData.AreaCntMap, "区域")
	}

	// 生成省份分布饼图
	if len(aggData.ProvCntMap) > 0 {
		chartsHTML += generatePieChartHTML(aggData.ProvCntMap, "省份")
	}

	if len(aggData.AreaLagCntMap) > 0 {
		chartsHTML += generatePieChartHTML(aggData.AreaLagCntMap, "区域延迟分布")
	}

	if len(aggData.ProvLagCntMap) > 0 {
		chartsHTML += generatePieChartHTML(aggData.ProvLagCntMap, "省份延迟分布")
	}

	return chartsHTML
}

// generatePieChartHTML 生成饼图HTML
func generatePieChartHTML(data map[string]int, title string) string {
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
func generateUpstreamDistributeChartsHTML(areaCntMap, provCntMap map[string]int) string {
	var chartsHTML string

	// 生成大区分布饼图
	if len(areaCntMap) > 0 {
		chartsHTML += generateUpstreamPieChartHTML(areaCntMap, "源站按大区分布")
	}

	// 生成省份分布饼图
	if len(provCntMap) > 0 {
		chartsHTML += generateUpstreamPieChartHTML(provCntMap, "源站按省分布")
	}

	return chartsHTML
}

// generateUpstreamPieChartHTML 生成源站分布饼图HTML
func generateUpstreamPieChartHTML(data map[string]int, title string) string {
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

func GenerateLineChart(title, seriesName, color string, xAxisData []string, yAxisData []opts.LineData) string {

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

	if color == "" {
		color = "#28a745"
	}

	// 添加数据系列
	line.SetXAxis(xAxisData).
		AddSeries(seriesName, yAxisData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: color,
				Width: 2,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color:   color,
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
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">%s</h2>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 450px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, title, encodedHTML)

	return htmlContent
}

// generateLagRateByStreamsTable 生成卡顿率按流分布表格
func GenerateLagRateByStreamsTable(data []util.HyLagRateByStreamsReport) string {
	if len(data) == 0 {
		return `
		<div class="chart-container">
			<h3>卡顿率按流分布</h3>
			<p>暂无数据</p>
		</div>`
	}

	// 准备表格数据
	var tableRows []string
	for _, item := range data {
		streamName := ""
		if item.StreamName != nil {
			streamName = *item.StreamName
		}

		percent := 0.0
		if item.Percent != nil {
			percent = *item.Percent
		}

		lagCount := 0
		if item.LagCnt != nil {
			lagCount = *item.LagCnt
		}

		total := 0
		if item.Total != nil {
			total = *item.Total
		}

		tableRows = append(tableRows, fmt.Sprintf(`{
			"streamName": "%s",
			"lagCount": %d,
			"totalCount": %d,
			"lagRate": "%.2f%%"
		}`, streamName, lagCount, total, percent))
	}

	tableData := strings.Join(tableRows, ",\n")

	// 生成包含AG Grid的HTML页面
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>卡顿率按流分布</title>
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
    <h2 style="text-align: center; color: #333; margin-bottom: 20px;">卡顿率按流分布</h2>
    <div id="grid-container">
        <div id="myGrid" class="ag-theme-alpine"></div>
    </div>

    <script>
        // 定义表格列配置
        const columnDefs = [
            {
                headerName: "流名称",
                field: "streamName",
                sortable: true,
                filter: true,
                width: 850
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
                headerName: "卡顿率",
                field: "lagRate",
                sortable: true,
                filter: 'agNumberColumnFilter',
                width: 120,
                comparator: (valueA, valueB) => {
                    const numA = parseFloat(valueA.replace('%%', ''));
                    const numB = parseFloat(valueB.replace('%%', ''));
                    return numA - numB;
                }
            }
        ];

        // 表格数据
        const rowData = [
            %s
        ];

        // AG Grid 配置
        const gridOptions = {
            columnDefs: columnDefs,
            rowData: rowData,
            pagination: true,
            paginationPageSize: 25,
            domLayout: 'normal',
            defaultColDef: {
                resizable: true,
                sortable: true,
                filter: true
            },
            enableCellTextSelection: true,
            ensureDomOrder: true
        };

        // 创建表格
        const gridDiv = document.querySelector('#myGrid');
        try {
            agGrid.createGrid(gridDiv, gridOptions);
            console.log('表格创建成功');
        } catch(error) {
            console.error('创建表格失败:', error);
        }
    </script>
</body>
</html>`, tableData)

	// 将完整HTML页面编码为Data URL
	encodedHTML := base64.StdEncoding.EncodeToString([]byte(htmlContent))

	// 使用iframe包装图表HTML作为web component
	return fmt.Sprintf(`
		<div style="margin-top: 40px; border-top: 2px solid #eee; padding-top: 30px;">
			<h2 style="text-align: center; color: #333; margin-bottom: 30px;">卡顿率按流分布</h2>
			<iframe 
				src="data:text/html;base64,%s" 
				style="width: 100%%; height: 700px; border: 1px solid #ddd; border-radius: 4px;"
				sandbox="allow-scripts allow-same-origin"
				frameborder="0"
			></iframe>
		</div>
	`, encodedHTML)
}

package qos

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// generateCDNLagTableHTML 生成CDN卡顿用户表格
func GenerateCDNLagTableHTML(cdnAggDatas []CdnAggregateData) string {
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

        // 创建主表格实例
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

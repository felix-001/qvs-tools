# MIKU QOS排障系统

## 功能概述

MIKU QOS排障系统是一个基于Web的质量分析工具，用于查询和分析流媒体质量数据。系统提供友好的Web界面，支持多种查询条件和实时数据分析。

## 主要功能

### 1. Web界面功能
- **时间范围选择**: 支持开始和结束日期时间的选择，也支持手动输入
- **应用名称筛选**: 通过下拉框选择不同的AppName
- **流ID查询**: 支持精确匹配和模糊查询
- **实时分析**: 点击"开始质量分析"按钮即可获取分析结果

### 2. 后端API接口
- `GET /` - 主页面
- `GET /api/v1/appnames` - 获取应用名称列表
- `POST /api/v1/qos` - 执行QOS分析

### 3. 数据展示
- 表格形式展示查询结果
- 支持时间、流ID、平台、IP等多维度信息展示
- 响应式设计，适配不同屏幕尺寸

## 快速开始

### 1. 编译程序
```bash
go build main_qos.go -o qos-server
```

### 2. 启动服务器
```bash
./qos-server
```

### 3. 访问系统
打开浏览器访问: http://localhost:8080

## 技术架构

### 前端技术栈
- HTML5 + CSS3 + JavaScript (原生)
- 响应式设计
- 异步请求 (Fetch API)

### 后端技术栈
- Go语言
- HTTP标准库
- Trino数据库连接
- SQL查询构建

### 数据存储
- Trino分布式查询引擎
- Hive数据仓库
- 质量报告表 (huyabiz_quality_report_log)

## 查询字段说明

### 可查询的字段
- **client_type**: 客户端类型
- **dim_ip**: IP地址
- **dim_isp**: ISP运营商
- **dim_cdndomain**: CDN域名
- **dim_cdnip**: CDN IP地址
- **dim_coderatebps**: 编码码率
- **dim_platform**: 平台信息
- **dim_stream**: 流ID
- **field_video_bad_quality**: 不良质量指标

### 查询条件
- **时间范围**: 支持开始和结束时间筛选
- **应用名称**: 精确匹配平台信息
- **流ID**: 支持精确和模糊匹配

## API接口文档

### 获取应用名称列表
**请求:** GET /api/v1/appnames

**响应:**
```json
["miku_live", "miku_vod", "miku_short", "miku_game", "miku_education"]
```

### 执行QOS分析
**请求:** POST /api/v1/qos

**请求体:**
```json
{
  "appName": "miku_live",
  "startTime": "2025-12-01T00:00",
  "endTime": "2025-12-03T23:59",
  "streamId": "stream123",
  "fuzzySearch": true
}
```

**响应:** HTML格式的分析报告

## 配置说明

### Trino连接配置
```go
dsn := "http://superset@trino.jf-logverse.k8s.qiniu.io?catalog=hive_miku&schema=miku"
```

### 端口配置
- HTTP服务端口: 8080
- 可在代码中修改

## 部署说明

### 开发环境
1. 确保Go环境版本 >= 1.20
2. 安装依赖: `go mod tidy`
3. 编译运行

### 生产环境
1. 编译为可执行文件
2. 配置systemd服务
3. 配置nginx反向代理（可选）

## 故障排除

### 常见问题

1. **端口被占用**
   - 检查8080端口是否被占用
   - 修改代码中的端口号

2. **Trino连接失败**
   - 检查网络连接
   - 验证Trino服务状态
   - 确认认证信息正确

3. **查询无数据**
   - 检查时间范围是否正确
   - 确认表名和字段名
   - 检查数据权限

### 日志信息
系统会在控制台输出详细的日志信息，包括：
- 查询SQL语句
- 数据库连接状态
- 错误信息

## 扩展功能

### 计划中的功能
- [ ] 数据导出功能 (CSV/Excel)
- [ ] 图表可视化
- [ ] 定时任务和报表
- [ ] 用户权限管理
- [ ] 查询历史记录

### 自定义扩展
- 修改HTML模板可调整界面样式
- 添加新的查询条件
- 集成其他数据源

## 联系支持

如有问题或建议，请联系开发团队。
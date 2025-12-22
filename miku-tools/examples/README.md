# 进程监控与邮件通知系统

这是一个用Go编写的进程监控系统，可以监测指定进程的重启，并通过邮件发送通知。

## 功能特点

- 监控指定进程的重启
- 支持多种邮件服务器配置（SMTP）
- 支持TLS加密连接
- 可配置检查间隔
- 支持同时监控多个进程
- 详细的邮件通知信息

## 文件说明

1. **qvs/mail.go** - 核心功能实现
   - 邮件发送功能
   - 进程状态检查
   - 进程监控逻辑

2. **examples/simple_monitor.go** - 简单示例
   - 最基本的使用方式
   - 适合快速测试和集成

3. **examples/process_monitor.go** - 命令行工具
   - 支持命令行参数配置
   - 包含邮件测试功能
   - 适合作为独立工具使用

4. **examples/config_monitor.go** - 配置文件驱动的监控程序
   - 支持JSON配置文件
   - 可同时监控多个进程
   - 每个进程可配置不同的收件人
   - 适合生产环境使用

5. **examples/monitor_config.json** - 配置文件示例
   - 展示配置文件格式
   - 包含SMTP和多进程配置

## 快速开始

### 1. 使用简单示例

修改 `examples/simple_monitor.go` 中的邮件配置和进程名称：

```go
mailConfig := &qvs.MailConfig{
    SMTPHost: "smtp.gmail.com",       // 你的SMTP服务器
    SMTPPort: 587,                    // SMTP端口
    Username: "your-email@gmail.com",  // 你的邮箱
    Password: "your-password",        // 你的密码或授权码
    From:     "your-email@gmail.com",  // 发件人邮箱
    To:       []string{"admin@example.com"},  // 收件人邮箱
    UseTLS:   true,                   // 使用TLS
}

processName := "nginx"  // 要监控的进程名
```

然后运行：

```bash
go run examples/simple_monitor.go
```

### 2. 使用命令行工具

```bash
go run examples/process_monitor.go \
  -process=nginx \
  -host=smtp.gmail.com \
  -port=587 \
  -user=your-email@gmail.com \
  -pass=your-password \
  -from=your-email@gmail.com \
  -to=admin@example.com \
  -tls=true \
  -interval=30s
```

### 3. 使用配置文件

1. 复制并修改配置文件：

```bash
cp examples/monitor_config.json my_config.json
```

2. 编辑 `my_config.json`，修改SMTP和进程配置：

```json
{
  "smtp": {
    "host": "smtp.gmail.com",
    "port": 587,
    "username": "your-email@gmail.com",
    "password": "your-password",
    "from": "your-email@gmail.com",
    "useTLS": true
  },
  "processes": [
    {
      "name": "nginx",
      "recipients": ["admin@example.com"],
      "checkInterval": "10s"
    },
    {
      "name": "mysql",
      "recipients": ["dba@example.com", "admin@example.com"],
      "checkInterval": "30s"
    }
  ]
}
```

3. 运行程序：

```bash
go run examples/config_monitor.go my_config.json
```

## 邮件服务器配置

以下是常见邮件服务器的配置信息：

### Gmail
- SMTP服务器: smtp.gmail.com
- 端口: 587 (TLS) 或 465 (SSL)
- 需要开启"两步验证"并使用"应用专用密码"

### Outlook/Hotmail
- SMTP服务器: smtp-mail.outlook.com
- 端口: 587 (TLS)

### 阿里云邮箱
- SMTP服务器: smtp.mxhichina.com
- 端口: 587 (TLS)

### 腾讯企业邮箱
- SMTP服务器: smtp.exmail.qq.com
- 端口: 587 (TLS)

## 注意事项

1. **进程检测**: 程序使用 `pgrep -f` 命令查找进程，所以进程名可以是完整路径或部分名称。

2. **系统兼容性**: 当前实现基于Linux的 `/proc` 文件系统，其他系统可能需要调整。

3. **安全性**: 邮件密码明文存储在代码或配置文件中，生产环境建议使用环境变量或加密存储。

4. **资源占用**: 监控多个进程时，每个进程会有一个独立的goroutine，资源占用相对较少。

## 扩展功能

可以根据需要扩展以下功能：

1. 添加进程CPU/内存使用率监控
2. 支持短信或其他通知方式
3. 添加Web管理界面
4. 支持进程自动重启
5. 添加监控历史记录和统计

## 故障排除

1. **邮件发送失败**: 检查SMTP服务器配置、用户名密码和防火墙设置。

2. **进程检测失败**: 确认进程名正确，且 `pgrep` 命令可用。

3. **权限问题**: 确保程序有权限访问 `/proc` 文件系统。

4. **依赖问题**: 确保Go环境正确配置，且所有依赖包已安装。
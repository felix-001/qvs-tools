package main

import (
	"fmt"
	"time"

	"mikutool/qvs"
)

func main() {
	// 配置邮件
	mailConfig := &qvs.MailConfig{
		SMTPHost: "smtp.example.com",  // 替换为你的SMTP服务器
		SMTPPort: 587,                 // SMTP端口
		Username: "your-email@example.com",  // 替换为你的邮箱
		Password: "your-password",     // 替换为你的密码或授权码
		From:     "your-email@example.com",  // 发件人邮箱
		To:       []string{"recipient@example.com"},  // 收件人邮箱
		UseTLS:   true,                // 使用TLS
	}
	
	// 要监控的进程名称
	processName := "nginx"  // 替换为你要监控的进程名
	
	// 创建进程监控器，每10秒检查一次
	monitor := qvs.NewProcessMonitor(processName, mailConfig, 10*time.Second)
	
	fmt.Printf("开始监控进程: %s\n", processName)
	
	// 开始监控(这是一个阻塞操作)
	monitor.MonitorProcess()
}
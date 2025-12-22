package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mikutool/qvs"
)

func main() {
	// 命令行参数
	processName := flag.String("process", "", "要监控的进程名称")
	smtpHost := flag.String("host", "smtp.example.com", "SMTP服务器地址")
	smtpPort := flag.Int("port", 587, "SMTP端口")
	username := flag.String("user", "", "邮箱用户名")
	password := flag.String("pass", "", "邮箱密码或授权码")
	from := flag.String("from", "", "发件人邮箱")
	to := flag.String("to", "", "收件人邮箱(多个用逗号分隔)")
	useTLS := flag.Bool("tls", true, "是否使用TLS")
	interval := flag.Duration("interval", 10*time.Second, "检查间隔")
	
	flag.Parse()
	
	// 检查必需参数
	if *processName == "" || *username == "" || *password == "" || *from == "" || *to == "" {
		fmt.Println("使用方法:")
		fmt.Println("  process_monitor -process=<进程名> -user=<邮箱> -pass=<密码> -from=<发件邮箱> -to=<收件邮箱>")
		fmt.Println("\n可选参数:")
		flag.PrintDefaults()
		os.Exit(1)
	}
	
	// 解析收件人列表
	recipients := strings.Split(*to, ",")
	for i, r := range recipients {
		recipients[i] = strings.TrimSpace(r)
	}
	
	// 创建邮件配置
	mailConfig := &qvs.MailConfig{
		SMTPHost: *smtpHost,
		SMTPPort: *smtpPort,
		Username: *username,
		Password: *password,
		From:     *from,
		To:       recipients,
		UseTLS:   *useTLS,
	}
	
	// 测试邮件配置
	fmt.Println("测试邮件配置...")
	subject := "进程监控系统启动测试"
	body := `
		<html>
		<body>
			<h2>进程监控系统测试</h2>
			<p>进程监控系统已启动，开始监控进程: ` + *processName + `</p>
			<p>启动时间: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
			<br>
			<p>此邮件由进程监控系统自动发送。</p>
		</body>
		</html>
	`
	
	if err := mailConfig.SendMail(subject, body); err != nil {
		log.Printf("测试邮件发送失败: %v", err)
		fmt.Println("警告: 邮件配置可能有问题，但监控将继续进行")
	} else {
		fmt.Println("测试邮件发送成功")
	}
	
	// 创建进程监控器
	monitor := qvs.NewProcessMonitor(*processName, mailConfig, *interval)
	
	// 设置信号处理，优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// 启动监控协程
	go func() {
		monitor.MonitorProcess()
	}()
	
	fmt.Println("进程监控系统已启动，按Ctrl+C退出...")
	<-sigChan
	fmt.Println("\n正在关闭进程监控系统...")
}
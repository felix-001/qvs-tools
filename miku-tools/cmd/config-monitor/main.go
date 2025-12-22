package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"mikutool/qvs"
)

// Config 整体配置结构
type Config struct {
	SMTP     SMTPConfig     `json:"smtp"`
	Processes []ProcessConfig `json:"processes"`
}

// SMTPConfig SMTP配置
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	UseTLS   bool   `json:"useTLS"`
}

// ProcessConfig 进程配置
type ProcessConfig struct {
	Name          string   `json:"name"`
	Recipients    []string `json:"recipients"`
	CheckInterval string   `json:"checkInterval"`
}

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("使用方法: config-monitor <配置文件路径>")
		os.Exit(1)
	}
	
	configFile := os.Args[1]
	
	// 读取配置文件
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}
	
	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}
	
	// 创建等待组，用于等待所有监控协程结束
	var wg sync.WaitGroup
	// 创建通道，用于接收退出信号
	stopChan := make(chan struct{})
	
	// 为每个进程创建监控器
	for _, procConfig := range config.Processes {
		// 解析检查间隔
		checkInterval, err := time.ParseDuration(procConfig.CheckInterval)
		if err != nil {
			log.Printf("解析检查间隔失败 [%s]: %v，使用默认值10秒", procConfig.Name, err)
			checkInterval = 10 * time.Second
		}
		
		// 创建邮件配置
		mailConfig := &qvs.MailConfig{
			SMTPHost: config.SMTP.Host,
			SMTPPort: config.SMTP.Port,
			Username: config.SMTP.Username,
			Password: config.SMTP.Password,
			From:     config.SMTP.From,
			To:       procConfig.Recipients,
			UseTLS:   config.SMTP.UseTLS,
		}
		
		// 创建进程监控器
		monitor := qvs.NewProcessMonitor(procConfig.Name, mailConfig, checkInterval)
		
		// 启动监控协程
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			fmt.Printf("开始监控进程: %s，检查间隔: %v\n", name, checkInterval)
			
			// 创建定时器，用于定期检查退出信号
			ticker := time.NewTicker(checkInterval)
			defer ticker.Stop()
			
			// 初始检查
			if running, pid, err := qvs.IsProcessRunning(name); err == nil && running {
				monitor.LastPID = pid
				fmt.Printf("进程 %s 当前运行中，PID: %d\n", name, pid)
			}
			
			for {
				select {
				case <-stopChan:
					fmt.Printf("停止监控进程: %s\n", name)
					return
				case <-ticker.C:
					// 检查进程状态
					running, pid, err := qvs.IsProcessRunning(name)
					if err != nil {
						fmt.Printf("检查进程状态出错 [%s]: %v\n", name, err)
						continue
					}
					
					if !running {
						if monitor.LastPID != 0 {
							// 进程已停止
							fmt.Printf("进程 %s 已停止\n", name)
							monitor.LastPID = 0
						}
					} else {
						if monitor.LastPID == 0 {
							// 进程新启动
							fmt.Printf("检测到进程 %s 重启，新PID: %d\n", name, pid)
							
							// 获取进程启动时间
							startTime, err := qvs.GetProcessStartTime(pid)
							if err != nil {
								fmt.Printf("获取进程启动时间失败 [%s]: %v\n", name, err)
							} else {
								// 发送邮件通知
								subject := fmt.Sprintf("进程重启通知: %s", name)
								body := fmt.Sprintf(`
									<html>
									<body>
										<h2>进程重启通知</h2>
										<p><strong>进程名称:</strong> %s</p>
										<p><strong>新PID:</strong> %d</p>
										<p><strong>启动时间:</strong> %s</p>
										<p><strong>检测时间:</strong> %s</p>
										<br>
										<p>此邮件由进程监控系统自动发送。</p>
									</body>
									</html>
								`, name, pid, startTime.Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02 15:04:05"))
								
								if err := monitor.MailConfig.SendMail(subject, body); err != nil {
									fmt.Printf("发送邮件失败 [%s]: %v\n", name, err)
								} else {
									fmt.Printf("已发送进程重启通知邮件 [%s]\n", name)
								}
							}
							
							monitor.LastPID = pid
						} else if pid != monitor.LastPID {
							// 进程PID改变，表示重启
							fmt.Printf("检测到进程 %s 重启，旧PID: %d，新PID: %d\n", name, monitor.LastPID, pid)
							
							// 获取进程启动时间
							startTime, err := qvs.GetProcessStartTime(pid)
							if err != nil {
								fmt.Printf("获取进程启动时间失败 [%s]: %v\n", name, err)
							} else {
								// 发送邮件通知
								subject := fmt.Sprintf("进程重启通知: %s", name)
								body := fmt.Sprintf(`
									<html>
									<body>
										<h2>进程重启通知</h2>
										<p><strong>进程名称:</strong> %s</p>
										<p><strong>旧PID:</strong> %d</p>
										<p><strong>新PID:</strong> %d</p>
										<p><strong>启动时间:</strong> %s</p>
										<p><strong>检测时间:</strong> %s</p>
										<br>
										<p>此邮件由进程监控系统自动发送。</p>
									</body>
									</html>
								`, name, monitor.LastPID, pid, startTime.Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02 15:04:05"))
								
								if err := monitor.MailConfig.SendMail(subject, body); err != nil {
									fmt.Printf("发送邮件失败 [%s]: %v\n", name, err)
								} else {
									fmt.Printf("已发送进程重启通知邮件 [%s]\n", name)
								}
							}
							
							monitor.LastPID = pid
						}
					}
				}
			}
		}(procConfig.Name)
	}
	
	// 设置信号处理，优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	fmt.Printf("进程监控系统已启动，共监控 %d 个进程，按Ctrl+C退出...\n", len(config.Processes))
	<-sigChan
	
	fmt.Println("\n正在关闭进程监控系统...")
	// 通知所有监控协程退出
	close(stopChan)
	
	// 等待所有监控协程结束
	wg.Wait()
	fmt.Println("进程监控系统已关闭")
}
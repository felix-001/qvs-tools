package qvs

import (
	"log"
	"middle-source-analysis/util"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func Perf() {
	// 每隔1秒检查一次
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// 用于存储perf record命令的进程
	var perfCmd *exec.Cmd

	for {
		select {
		case <-ticker.C:
			// 获取srs进程的CPU占用率
			cpuUsage, err := getSRSCPUUsage()
			if err != nil {
				log.Printf("Error getting CPU usage: %v", err)
				continue
			}

			// 如果CPU占用率大于70%，执行perf record命令
			if cpuUsage > 70 {
				if perfCmd == nil || perfCmd.Process == nil {
					perfCmd = exec.Command("perf", "record", "-p", "$(pidof srs)")
					if err := perfCmd.Start(); err != nil {
						log.Printf("Error starting perf record: %v", err)
						continue
					}
					log.Println("perf record started")
				}

				// 等待3分钟
				time.Sleep(3 * time.Minute)

				// 终止perf record命令
				if perfCmd.Process != nil {
					if err := perfCmd.Process.Kill(); err != nil {
						log.Printf("Error killing perf record: %v", err)
					} else {
						log.Println("perf record killed")
					}
				}

				// 发送告警到企业微信
				util.SendWeChatAlert("srs process CPU usage exceeded 70%")

				// 退出程序
				return
			}
		}
	}
}

// getSRSCPUUsage 获取srs进程的CPU占用率
func getSRSCPUUsage() (float64, error) {
	// 使用top命令获取srs进程的CPU占用率
	cmd := exec.Command("sh", "-c", "top -b -n 1 | grep srs | awk '{print $9}'")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// 将输出转换为浮点数
	cpuUsage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}

	return cpuUsage, nil
}

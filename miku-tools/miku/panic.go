package miku

import (
	"fmt"
	"log"
	"mikutool/config"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	panicDir      = "/home/qboxserver/miku-sched/_package/run"
	localBackup   = "/tmp/sched-panic-log"
	checkInterval = 1 * time.Minute
)

var nodes = []string{"vdn-jsyz1-dls-1-31", "xs3431", "xs3430", "bili-xs11", "xs4664", "bili-xs13", "vdn-jsyz1-dls-1-30", "bili-xs12", "bili-xs16", "xs4834", "vdn-jsyz1-dls-1-29", "bili-xs9", "vdn-jsyz1-dls-1-64", "bili-xs8", "xs3427", "vdn-jsyz1-dls-1-78", "vdn-jsyz1-dls-1-79"}

// alertedNodes 记录已发送告警的节点，避免重复告警
var alertedNodes sync.Map

// TracePanic 定时检查各节点 miku-sched 的 panic 日志，
// 发现 panic 则备份文件并发送企业微信告警
func (m *Miku) TracePanic(cfg *config.Config) {
	if m.conf.Raw == "" {
		log.Println("[TracePanic] 没有配置 -raw，跳过")
		return
	}
	if len(nodes) == 0 {
		log.Println("[TracePanic] 没有配置 sched_panic_nodes，跳过")
		return
	}

	wecom := NewWeComRobot(cfg.WeComWebhook)

	log.Printf("[TracePanic] 启动定时检查，间隔 %v，节点: %v", checkInterval, nodes)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// 首次立即执行一次
	m.checkAndAlert(nodes, wecom)

	for range ticker.C {
		m.checkAndAlert(nodes, wecom)
	}
}

// checkAndAlert 对每个节点执行检查和告警
func (m *Miku) checkAndAlert(nodes []string, wecom *WeComRobot) {
	for _, node := range nodes {
		node = strings.TrimSpace(node)
		if node == "" {
			continue
		}
		m.processNode(node, wecom)
	}
}

// processNode 处理单个节点的 panic 检查
func (m *Miku) processNode(node string, wecom *WeComRobot) {
	// 1. 通过 qssh 检查远程节点是否有 panic 日志
	grepCmd := fmt.Sprintf("grep -rl %s %s", m.conf.Raw, panicDir)
	log.Println("grepCmd:", grepCmd)
	cmd := exec.Command("qssh", node, grepCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// grep 返回非零退出码（没有匹配到）是正常情况，不是错误
		// 所以只有当输出非空时才需要处理
		if len(output) == 0 {
			return
		}
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		// 没有 panic 日志
		return
	}

	log.Printf("[TracePanic] 节点 %s 发现 panic 日志", node)

	// 2. 检查是否已经告警过（避免重复）
	alertedKey := node + "_" + time.Now().Format("2006-01-02")
	if _, already := alertedNodes.Load(alertedKey); already {
		log.Printf("[TracePanic] 节点 %s 今天已告警过，跳过", node)
		return
	}

	// 3. 备份远程 run 目录到远程 /tmp
	if err := m.backupRemoteDir(node, localBackup); err != nil {
		log.Printf("[TracePanic] 备份节点 %s 文件失败: %v", node, err)
		return
	}

	log.Printf("[TracePanic] 节点 %s 备份完成到 %s", node, localBackup)

	// 4. 发送企业微信告警
	if wecom != nil && wecom.WebhookURL != "" {
		// 截取 panic 日志的前几行作为告警内容
		panicLog := outputStr
		lines := strings.Split(panicLog, "\n")
		if len(lines) > 10 {
			panicLog = strings.Join(lines[:10], "\n")
			panicLog += "\n..."
		}

		if err := wecom.SendPanicAlert(node, panicLog); err != nil {
			log.Printf("[TracePanic] 发送告警失败: %v", err)
		} else {
			alertedNodes.Store(alertedKey, true)
		}
	}
}

// backupRemoteDir 在远程节点上清空旧目录后，将 run 目录备份到 /tmp
func (m *Miku) backupRemoteDir(node, remoteDir string) error {
	backupCmd := fmt.Sprintf("rm -rf %s && mkdir -p %s && cp -r %s/* %s/", remoteDir, remoteDir, panicDir, remoteDir)
	log.Println("backupCmd:", backupCmd)
	cmd := exec.Command("qssh", node, backupCmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("远程备份失败: %v, stderr: %s", err, stderr.String())
	}
	return nil
}

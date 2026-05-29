package miku

import (
	"fmt"
	"log"
	"mikutool/config"
	"os/exec"
	"strings"
	"sync"
)

var nginxNodes = []string{
	"vdn-jsyz1-dls-1-67",
	"vdn-jsyz1-dls-1-66",
	"vdn-jsyz1-dls-1-65",
}

// NginxLogSearch 在多个节点上并行搜索 nginx 日志
func (m *Miku) NginxLogSearch(cfg *config.Config) {
	if cfg.Path == "" {
		log.Fatalf("[NginxLogSearch] -path 不能为空")
	}
	if cfg.Pattern == "" {
		log.Fatalf("[NginxLogSearch] -pattern 不能为空")
	}
	if cfg.Query == "" {
		log.Fatalf("[NginxLogSearch] -query 不能为空")
	}

	log.Printf("[NginxLogSearch] 开始在 %d 个节点上搜索日志，path=%s, pattern=%s, query=%s",
		len(nginxNodes), cfg.Path, cfg.Pattern, cfg.Query)

	var wg sync.WaitGroup
	for _, node := range nginxNodes {
		node := strings.TrimSpace(node)
		if node == "" {
			continue
		}
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			searchNodeLog(n, cfg.Path, cfg.Pattern, cfg.Query)
		}(node)
	}
	wg.Wait()
	log.Println("[NginxLogSearch] 所有节点搜索完成")
}

func searchNodeLog(node, path, pattern, query string) {
	filePattern := path + "/" + pattern
	grepCmd := fmt.Sprintf("grep '%s' %s", query, filePattern)
	log.Printf("[%s] 执行命令: %s", node, grepCmd)
	cmd := exec.Command("qssh", node, grepCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// grep 返回非零退出码可能是没有匹配到
		if len(output) == 0 {
			log.Printf("[%s] 未找到匹配结果", node)
			return
		}
	}
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		log.Printf("[%s] 未找到匹配结果", node)
		return
	}
	log.Printf("[%s] 搜索结果:\n%s", node, outputStr)
}

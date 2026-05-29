package miku

import (
	"bufio"
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

// fileTarget 表示远程节点上的一个文件
type fileTarget struct {
	node string
	path string
}

// NginxLogSearch 先在每个节点上列出所有匹配的文件，再对所有文件并行搜索
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

	log.Printf("[NginxLogSearch] 开始，path=%s, pattern=%s, query=%s",
		cfg.Path, cfg.Pattern, cfg.Query)

	// 第一阶段：在每个节点上 ls 匹配的文件
	var mu sync.Mutex
	var files []fileTarget
	var wg sync.WaitGroup

	for _, node := range nginxNodes {
		node := strings.TrimSpace(node)
		if node == "" {
			continue
		}
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			matches, err := listFilesOnNode(n, cfg.Path, cfg.Pattern)
			if err != nil {
				log.Printf("[%s] 列出文件失败: %v", n, err)
				return
			}
			mu.Lock()
			for _, f := range matches {
				files = append(files, fileTarget{node: n, path: f})
			}
			mu.Unlock()
		}(node)
	}
	wg.Wait()

	if len(files) == 0 {
		log.Println("[NginxLogSearch] 未找到任何匹配的文件")
		return
	}

	log.Printf("[NginxLogSearch] 共找到 %d 个文件，开始并行搜索", len(files))

	// 第二阶段：对每个文件并行执行 grep
	var searchWg sync.WaitGroup
	for _, ft := range files {
		searchWg.Add(1)
		go func(ft fileTarget) {
			defer searchWg.Done()
			grepFileOnNode(ft.node, ft.path, cfg.Query)
		}(ft)
	}
	searchWg.Wait()
	log.Println("[NginxLogSearch] 所有节点、所有文件搜索完成")
}

// listFilesOnNode 在远程节点上执行 ls 列出所有匹配 path/pattern 的文件
func listFilesOnNode(node, path, pattern string) ([]string, error) {
	filePattern := path + "/" + pattern
	lsCmd := fmt.Sprintf("ls %s 2>/dev/null", filePattern)
	log.Printf("[%s] 列出文件: %s", node, lsCmd)
	cmd := exec.Command("qssh", node, lsCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ls 失败: %v, output: %s", err, string(output))
	}
	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}
	log.Printf("[%s] 找到 %d 个文件", node, len(files))
	return files, nil
}

// grepFileOnNode 在远程节点的指定文件上执行 grep
func grepFileOnNode(node, filePath, query string) {
	grepCmd := fmt.Sprintf("grep -E '%s' %s", query, filePath)
	log.Printf("[%s] 搜索文件 %s", node, filePath)
	log.Printf("[%s] 执行命令: %s", node, grepCmd)
	cmd := exec.Command("qssh", node, grepCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			log.Printf("[%s] 文件 %s：未找到匹配结果", node, filePath)
			return
		}
	}
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		log.Printf("[%s] 文件 %s：未找到匹配结果", node, filePath)
		return
	}
	log.Printf("[%s] 文件 %s 搜索结果:\n%s", node, filePath, outputStr)
}

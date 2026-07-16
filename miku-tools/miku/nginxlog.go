package miku

import (
	"bufio"
	"fmt"
	"log"
	"mikutool/config"
	"os"
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

// nodeSearchTarget 表示一个待搜索的节点及其日志路径/文件模式
type nodeSearchTarget struct {
	node    string
	path    string
	pattern string
}

// loadNginxNodes 优先从 /tmp/nodes.txt 读取节点列表。
// 每行格式: <nodeid>_<序号>，序号可为空；按序号生成 path/pattern。
// 读取失败时回退到内置 nginxNodes，使用 cfg.Path / cfg.Pattern。
func loadNginxNodes(cfg *config.Config) []nodeSearchTarget {
	data, err := os.ReadFile("/tmp/nodes.txt")
	if err != nil {
		log.Printf("[NginxLogSearch] 读取 /tmp/nodes.txt 失败，使用内置节点列表: %v", err)
		if cfg.Path == "" {
			log.Fatalf("[NginxLogSearch] -path 不能为空")
		}
		if cfg.Pattern == "" {
			log.Fatalf("[NginxLogSearch] -pattern 不能为空")
		}
		var targets []nodeSearchTarget
		for _, node := range nginxNodes {
			node = strings.TrimSpace(node)
			if node == "" {
				continue
			}
			targets = append(targets, nodeSearchTarget{
				node:    node,
				path:    cfg.Path,
				pattern: cfg.Pattern,
			})
		}
		return targets
	}

	var targets []nodeSearchTarget
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "_", 2)
		nodeId := parts[0]
		seq := ""
		if len(parts) > 1 {
			seq = parts[1]
		}
		// 序号为空 -> qvs-sip；序号为 2 -> qvs-sip2
		targets = append(targets, nodeSearchTarget{
			node:    nodeId,
			path:    "/home/qboxserver/qvs-sip" + seq + "/_package/run/",
			pattern: "qvs-sip" + seq + ".log*",
		})
	}
	log.Printf("[NginxLogSearch] 从 /tmp/nodes.txt 读取到 %d 个节点", len(targets))
	return targets
}

// NginxLogSearch 先在每个节点上列出所有匹配的文件，再对所有文件并行搜索
func (m *Miku) NginxLogSearch(cfg *config.Config) {
	if cfg.Query == "" {
		log.Fatalf("[NginxLogSearch] -query 不能为空")
	}

	targets := loadNginxNodes(cfg)
	if len(targets) == 0 {
		log.Println("[NginxLogSearch] 无可用节点")
		return
	}

	log.Printf("[NginxLogSearch] 开始，nodes=%d, query=%s", len(targets), cfg.Query)

	// 第一阶段：在每个节点上 ls 匹配的文件
	var mu sync.Mutex
	var files []fileTarget
	var wg sync.WaitGroup

	for _, t := range targets {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("[%s] path=%s, pattern=%s", t.node, t.path, t.pattern)
			matches, err := listFilesOnNode(t.node, t.path, t.pattern)
			if err != nil {
				log.Printf("[%s] 列出文件失败: %v", t.node, err)
				return
			}
			mu.Lock()
			for _, f := range matches {
				files = append(files, fileTarget{node: t.node, path: f})
			}
			mu.Unlock()
		}()
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

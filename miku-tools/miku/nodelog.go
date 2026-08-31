package miku

import (
	"fmt"
	"log"
	"mikutool/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	nodeLogBinaryDir = "/home/qboxserver/liyq"
	nodeLogBinary    = nodeLogBinaryDir + "/miku"
)

func NodeLog(cfg *config.Config) {
	nodes := splitSipSearchArg(cfg.Node)
	paths, err := expandNodeLogPaths(cfg.Path, cfg.PathParams)
	if err != nil {
		log.Printf("[NodeLog] %v", err)
		return
	}
	patterns := splitSipSearchArg(cfg.Pattern)
	if len(nodes) == 0 || len(paths) == 0 || len(patterns) == 0 {
		log.Println("[NodeLog] -node、-path、-path_params 和 -pattern 不能为空")
		return
	}

	archive := cfg.Archive
	if archive == "" {
		archive = "miku-1788147017.tar.gz"
	}
	if filepath.Base(archive) != archive {
		log.Printf("[NodeLog] -archive 只能传文件名: %s", archive)
		return
	}
	filePattern := cfg.Raw
	if filePattern == "" {
		filePattern = ".*dump.*log.*"
	}

	var outputMu sync.Mutex
	var outputFile *os.File
	if cfg.Output != "" {
		outputFile, err = os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("[NodeLog] 打开输出文件失败: %v", err)
			return
		}
		defer outputFile.Close()
	}

	var wg sync.WaitGroup
	for _, node := range nodes {
		node := node
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := runNodeLog(node, paths, strings.Join(patterns, ","), cfg.ExcludePattern, filePattern, archive, cfg.Force)
			outputMu.Lock()
			defer outputMu.Unlock()
			if outputFile != nil {
				if _, err := outputFile.WriteString(result); err != nil {
					log.Printf("[NodeLog] 写入输出文件失败: %v", err)
				}
			} else {
				fmt.Print(result)
			}
		}()
	}
	wg.Wait()
	log.Println("[NodeLog] 所有节点处理完成")
}

func expandNodeLogPaths(pathTemplate, pathParams string) ([]string, error) {
	if strings.TrimSpace(pathTemplate) == "" || strings.TrimSpace(pathParams) == "" {
		return nil, fmt.Errorf("-path 和 -path_params 不能为空")
	}
	parts := splitSipSearchArg(pathParams)
	if len(parts) == 0 {
		return nil, fmt.Errorf("-path_params 不能为空")
	}
	keyValue := strings.SplitN(parts[0], "=", 2)
	if len(keyValue) != 2 || strings.TrimSpace(keyValue[0]) == "" {
		return nil, fmt.Errorf("-path_params 格式应为 service=qvs-sip,qvs-sip1")
	}
	key := strings.TrimSpace(keyValue[0])
	values := append([]string{strings.TrimSpace(keyValue[1])}, parts[1:]...)
	for i, value := range values {
		if value == "" {
			return nil, fmt.Errorf("-path_params 包含空服务名")
		}
		values[i] = value
	}
	placeholder := "${" + key + "}"
	if !strings.Contains(pathTemplate, placeholder) {
		return []string{pathTemplate}, nil
	}
	paths := make([]string, 0, len(values))
	for _, value := range values {
		paths = append(paths, strings.ReplaceAll(pathTemplate, placeholder, value))
	}
	return paths, nil
}

func runNodeLog(node string, paths []string, pattern, excludePattern, filePattern, archive string, force bool) string {
	pathArg := strings.Join(paths, ",")
	prepare := fmt.Sprintf("mkdir -p %s && wget -q -O %s/%s %s && tar -xzf %s/%s -C %s", shellQuote(nodeLogBinaryDir), shellQuote(nodeLogBinaryDir), shellQuote(archive), shellQuote("http://qupfile.cloudvdn.com/"+archive), shellQuote(nodeLogBinaryDir), shellQuote(archive), shellQuote(nodeLogBinaryDir))
	if !force {
		prepare = fmt.Sprintf("if [ ! -f %s ]; then %s; fi", shellQuote(nodeLogBinary), prepare)
	}
	configFile := shellQuote("/tmp/mikutool.yaml")
	ensureConfig := fmt.Sprintf("if [ ! -f %s ]; then echo '{}' > %s; fi", configFile, configFile)
	command := fmt.Sprintf("%s && chmod +x %s && %s && %s -cmd sipsearch -path %s -pattern %s -exclude_pattern %s -raw %s", prepare, shellQuote(nodeLogBinary), ensureConfig, shellQuote(nodeLogBinary), shellQuote(pathArg), shellQuote(pattern), shellQuote(excludePattern), shellQuote(filePattern))
	output, err := exec.Command("qssh", node, command).CombinedOutput()
	result := fmt.Sprintf("[NodeLog] node=%s paths=%s\n%s", node, strings.Join(paths, ","), output)
	if err != nil {
		result += fmt.Sprintf("[NodeLog] node=%s 执行失败: %v\n", node, err)
	}
	return result
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

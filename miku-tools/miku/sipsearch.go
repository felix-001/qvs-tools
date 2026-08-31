package miku

import (
	"bufio"
	"fmt"
	"log"
	"mikutool/config"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const sipSearchDelimiter = "<--------------------------------------------------------------------------------------------------->"

func SipSearch(cfg *config.Config) {
	paths := splitSipSearchArg(cfg.Path)
	patterns := splitSipSearchArg(cfg.Pattern)
	if len(paths) == 0 {
		log.Println("[SipSearch] -path 不能为空")
		return
	}
	if len(patterns) == 0 {
		log.Println("[SipSearch] -pattern 不能为空")
		return
	}

	compiledPatterns := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("[SipSearch] 正则无效 %q: %v", pattern, err)
			return
		}
		compiledPatterns[i] = compiled
	}

	filePattern, err := compileSipSearchFilePattern(cfg.Raw)
	if err != nil {
		log.Printf("[SipSearch] 文件名正则无效 %q: %v", cfg.Raw, err)
		return
	}

	files := collectSipSearchFiles(paths, filePattern)
	if len(files) == 0 {
		log.Println("[SipSearch] 未找到符合条件的普通文件")
		return
	}
	log.Printf("[SipSearch] 找到 %d 个文件，开始处理", len(files))

	var wg sync.WaitGroup
	var outputMu sync.Mutex
	semaphore := make(chan struct{}, 10)
	for _, filePath := range files {
		filePath := filePath
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			searchSipSearchFile(filePath, compiledPatterns, &outputMu)
		}()
	}
	wg.Wait()
	log.Println("[SipSearch] 所有文件处理完成")
}

func splitSipSearchArg(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func compileSipSearchFilePattern(pattern string) (*regexp.Regexp, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, nil
	}
	return regexp.Compile(pattern)
}

func collectSipSearchFiles(paths []string, filePattern *regexp.Regexp) []string {
	var files []string
	seen := make(map[string]struct{})
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			log.Printf("[SipSearch] 访问路径 %s 失败: %v", root, err)
			continue
		}
		if !info.IsDir() {
			if _, exists := seen[root]; exists {
				continue
			}
			if filePattern == nil || filePattern.MatchString(filepath.Base(root)) {
				seen[root] = struct{}{}
				files = append(files, root)
			}
			continue
		}
		walkRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			log.Printf("[SipSearch] 解析路径 %s 失败: %v", root, err)
			continue
		}
		err = filepath.Walk(walkRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				log.Printf("[SipSearch] 访问路径 %s 失败: %v", path, walkErr)
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if filePattern != nil && !filePattern.MatchString(info.Name()) {
				return nil
			}
			if _, exists := seen[path]; !exists {
				seen[path] = struct{}{}
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			log.Printf("[SipSearch] 遍历路径 %s 失败: %v", root, err)
		}
	}
	return files
}

func searchSipSearchFile(filePath string, patterns []*regexp.Regexp, outputMu *sync.Mutex) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("[SipSearch] 读取文件 %s 失败: %v", filePath, err)
		return
	}

	matched := make([]bool, len(patterns))
	var record []string
	lineCount := 0
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		record = append(record, line)
		for i, pattern := range patterns {
			if pattern.MatchString(line) {
				matched[i] = true
			}
		}
		if strings.Contains(line, sipSearchDelimiter) {
			allMatched := true
			for _, isMatched := range matched {
				if !isMatched {
					allMatched = false
					break
				}
			}
			if allMatched {
				outputMu.Lock()
				fmt.Printf("[SipSearch] 文件 %s 命中信令:\n%s\n", filePath, strings.Join(record, "\n"))
				outputMu.Unlock()
			}
			record = record[:0]
			matched = make([]bool, len(patterns))
		}
		if lineCount%100000 == 0 {
			log.Printf("[SipSearch] 文件 %s 已处理 %d 行", filePath, lineCount)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[SipSearch] 处理文件 %s 失败: %v", filePath, err)
	}
	log.Printf("[SipSearch] 文件处理完成: %s，共 %d 行", filePath, lineCount)
}

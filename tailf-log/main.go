package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Logger struct {
	LastLogFile       string
	Cmd               *exec.Cmd
	LastChildCtx      context.Context
	LastChildCancel   context.CancelFunc
	LogFileNamePrefix string
	Regex             string
}

func (s *Logger) findLatestFile(dir string) (string, error) {
	var latestFile string
	latestTime := time.Unix(0, 0) // 初始化为 Unix 纪元时间，即1970-01-01 00:00:00 +0000 UTC

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		//log.Println(path)
		// 忽略目录
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".swp" {
			return nil
		}

		if !strings.HasPrefix(path, s.LogFileNamePrefix) {
			return nil
		}

		// 检查文件修改时间，并更新最新的文件信息
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = path
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if latestFile == "" {
		return "", fmt.Errorf("no files found in directory")
	}

	return latestFile, nil
}

func (s *Logger) Run(dir string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		filename, err := s.findLatestFile(dir)
		if err != nil {
			log.Printf("dir: %s, err: %v\n", dir, err)
			return
		}
		if s.LastLogFile == filename {
			time.Sleep(time.Second)
			continue
		}
		log.Printf("latest file: %s\n", filename)
		if s.LastChildCancel != nil {
			s.LastChildCancel()
			<-ctx.Done()
		}
		s.LastChildCtx, s.LastChildCancel = context.WithCancel(context.Background())
		go func(filename string, childCtx context.Context) {
			file, err := os.Open(filename)
			if err != nil {
				log.Fatalf("Open file fail:%v", err)
			}
			defer file.Close()
			reader := bufio.NewReader(file)

			for {
				select {
				case <-childCtx.Done():
					cancel()
					log.Println(filename, "quit")
					return
				default:
					line, err := reader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							time.Sleep(200 * time.Millisecond)
						} else {
							log.Println("err")
							break
						}
					}
					fmt.Print(string(line))
				}

			}

		}(filename, s.LastChildCtx)

		s.LastLogFile = filename
		time.Sleep(time.Second)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if len(os.Args) != 3 {
		log.Println("logger <service> <regex>")
		return
	}
	dir := ""
	logFileNamePrefix := ""
	switch os.Args[1] {
	case "sched":
		dir = "/home/qboxserver/miku-sched/_package/run"
		logFileNamePrefix = "miku-sched.log-"
	case "lived":
		dir = "/home/qboxserver/miku-lived/_package/run"
		logFileNamePrefix = "miku-lived.log-"
	case "monitor":
		dir = "/home/qboxserver/miku-monitor/_package/run"
		logFileNamePrefix = "miku-monitor.log-"
	default:
		dir = os.Args[1]
		logFileNamePrefix = "miku-sched.log-"
	}
	logger := &Logger{LogFileNamePrefix: logFileNamePrefix, Regex: os.Args[2]}
	logger.Run(dir)
}

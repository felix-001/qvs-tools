package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Logger struct {
	LastLogFile string
	Cmd         *exec.Cmd
}

func (s *Logger) findLatestFile(dir string) (string, error) {
	var latestFile string
	latestTime := time.Unix(0, 0) // 初始化为 Unix 纪元时间，即1970-01-01 00:00:00 +0000 UTC

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略目录
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".swp" {
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
	for {
		file, err := s.findLatestFile(dir)
		if err != nil {
			log.Printf("dir: %s, err: %v\n", dir, err)
			continue
		}
		if s.LastLogFile == file {
			time.Sleep(time.Second)
			continue
		}
		log.Printf("latest file: %s\n", file)
		if s.Cmd != nil {
			if err := s.Cmd.Process.Kill(); err != nil {
				log.Println("Error killing command:", err)
			} else {
				log.Printf("kill old success, %s\n", s.LastLogFile)
			}
		}
		go func() {
			file, err := os.Open("./new.txt") //针对test.log文件
			if err != nil {
				log.Fatalf("Open file fail:%v", err)
			}
			defer file.Close()
			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						time.Sleep(100 * time.Millisecond)
					} else {
						log.Println("err")
						break
					}
				}
				fmt.Print(string(line))
			}

			/*
				s.Cmd = exec.Command("tail", "-f", file)
				output, err := s.Cmd.StdoutPipe()
				if err != nil {
					log.Fatal(err)
				}
				defer output.Close()

				// 启动命令
				if err := s.Cmd.Start(); err != nil {
					log.Fatal(err)
				}

				bigreader := bufio.NewReader(output)
				line, isPrefix, err := bigreader.ReadLine()
				for err == nil && !isPrefix {
					log.Println(string(line))
					line, isPrefix, err = bigreader.ReadLine()
				}

				// 等待命令结束
				if err := s.Cmd.Wait(); err != nil {
					log.Println(err, file)
				}
				log.Printf("file %s end", file)
			*/
		}()
		s.LastLogFile = file
		time.Sleep(time.Second)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if len(os.Args) != 2 {
		log.Println("logger <service>")
		return
	}
	dir := ""
	switch os.Args[1] {
	case "sched":
		dir = "/home/qboxserver/miku-sched/_package/run"
	case "lived":
		dir = "/home/qboxserver/miku-lived/_package/run"
	case "monitor":
		dir = "/home/qboxserver/miku-monitor/_package/run"
	default:
		dir = os.Args[1]
	}
	logger := &Logger{}
	logger.Run(dir)
}

func main1() {
	file, err := os.Open("./new.txt") //针对test.log文件
	if err != nil {
		log.Fatalf("Open file fail:%v", err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
			} else {
				break
			}
		}
		fmt.Print(string(line))
	}
}

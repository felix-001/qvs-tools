package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

func RunCmd(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func parseFileList(raw string) []string {
	lines := strings.Split(raw, "\n")
	var out []string
	for _, line := range lines {
		if strings.Contains(line, ".log") {
			re := regexp.MustCompile(`\S+$`) // 匹配非空白字符的结尾
			result := re.FindString(line)
			out = append(out, result)
		}
	}
	return out
}

func getTimeOfLog(line string) (string, error) {
	start := strings.Index(line, "time=")
	if start == -1 {
		return "", fmt.Errorf("find time= err, %s", line)
	}
	line = line[start+len("time="):]
	end := strings.Index(line, "level=")
	if end == -1 {
		return "", fmt.Errorf("find level= err, %s", line)
	}
	line = line[:end-1]
	return line, nil
}

func sortLogByTime(txt string) string {
	lines := strings.Split(txt, "\n")
	sort.Slice(lines, func(i, j int) bool {
		t, err := getTimeOfLog(lines[i])
		if err != nil {
			log.Println(err)
			return false
		}
		t1, err := getTimeOfLog(lines[j])
		if err != nil {
			log.Println(err)
			return false
		}
		return t < t1
	})
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}
	return out
}

func fetchLog10Min(date, t, str, path string, idx int, f *os.File) {
	cmd := fmt.Sprintf("hdfs dfs -ls /logs-v2/*/%s/%s/%s", date, path, t)
	s, err := RunCmd(cmd)
	if err != nil {
		log.Println(err, cmd)
		return
	}
	//log.Println(s)
	files := parseFileList(s)
	//log.Println(files)
	var wg sync.WaitGroup

	wg.Add(len(files))

	for _, file := range files {
		go func(file string) {
			defer wg.Done()
			defer func() {
				count++
				log.Printf("progress: %d/%d %d/6\n", count, len(files), idx)
			}()
			cmd := fmt.Sprintf("hdfs dfs -cat %s | grep -E \"%s\"", file, str)
			//log.Println("run cmd", cmd)
			s, err := RunCmd(cmd)
			if err != nil {
				//log.Println(err, cmd)
				return
			}
			_, err = f.WriteString(s)
			if err != nil {
				fmt.Println("写入文件失败:", err)
				return
			}
			//log.Println("cmd", cmd, "done")
		}(file)
	}
	wg.Wait()
}

type WeComNotfiy struct {
	Msgtype  string `json:"msgtype"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

func wecomNotify(content string) {
	notify := WeComNotfiy{
		Msgtype: "markdown",
		Markdown: struct {
			Content string `json:"content"`
		}{Content: content},
	}
	body, err := json.Marshal(&notify)
	if err != nil {
		log.Println(err)
	}
	resp, err := http.Post("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=779b4815-6369-451b-a817-56189c97b549", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Println(err)
	}
	if resp.StatusCode != 200 {
		log.Println("http code err:", resp.StatusCode, resp.Status)
	}
}

var count int = 0

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var typ, date, t, str string
	var sort bool
	flag.StringVar(&typ, "typ", "log", "要查找的日志类型:log/audit")
	flag.StringVar(&date, "date", "", "日期(STREAMD_LOG_DATE),例如: 2024-05-31")
	flag.StringVar(&t, "t", "", "时间(STREAMD_LOG_T),例如: 20-20")
	flag.StringVar(&str, "str", "", "要搜索的字符串,例如: test")
	flag.BoolVar(&sort, "sort", false, "是否需要对结果排序")
	flag.Parse()
	dateEnv := os.Getenv("STREAMD_LOG_DATE")
	if dateEnv != "" {
		date = dateEnv
	}
	tEnv := os.Getenv("STREAMD_LOG_T")
	if tEnv != "" {
		t = tEnv
	}
	start := time.Now()
	path := "NGXQNM_MIKU-STREAMD"
	if typ == "audit" {
		path = "FLOWD_MIKU-STREAMD"
	}
	searchStr := strings.ReplaceAll(str, " ", "-")
	searchStr = strings.ReplaceAll(searchStr, ".", "-")
	searchStr = strings.ReplaceAll(searchStr, "*", "-")
	searchStr = strings.ReplaceAll(searchStr, "=", "-")
	searchStr = strings.ReplaceAll(searchStr, ":", "-")
	fileName := fmt.Sprintf("%s-%s-%s-%d.log", searchStr, date, t, time.Now().Unix())
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println("无法创建文件:", err)
		return
	}
	defer f.Close()
	if !strings.Contains(t, "*") {
		fetchLog10Min(date, t, str, path, 0, f)
	} else {
		ss := strings.Split(t, "-")
		if len(ss) != 2 {
			log.Println("parse time err", t)
			return
		}
		for i := 0; i < 6; i++ {
			t = fmt.Sprintf("%s-%02d", ss[0], i*10)
			fetchLog10Min(date, t, str, path, i, f)
		}
	}

	/*
		if sort {
			result = sortLogByTime(result)
		}
	*/
	//log.Println(result)
	/*
		err = ioutil.WriteFile("out.log", []byte(result), 0644)
		if err != nil {
			log.Println(err)
		}
	*/
	log.Println("cost", time.Now().Sub(start))
	content := fmt.Sprintf("search streamd log done, cost: %+v <@liyuanquan>", time.Now().Sub(start))
	wecomNotify(content)
}

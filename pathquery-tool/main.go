package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/qbox/mikud-live/common/model"
)

func splitIgnoringQuotes(input string) []string {
	var result []string
	var currentPart strings.Builder
	inQuotes := false

	for _, char := range input {
		switch char {
		case '"':
			inQuotes = !inQuotes // 切换引号状态
		case ',':
			if !inQuotes { // 只有在非引号内才进行分割
				result = append(result, currentPart.String())
				currentPart.Reset()
			} else {
				currentPart.WriteRune(char) // 引号内保留逗号
			}
		default:
			currentPart.WriteRune(char)
		}
	}

	// 添加最后一个部分
	if currentPart.Len() > 0 {
		result = append(result, currentPart.String())
	}

	return result
}

func findStr(line, startStr, endStr string) (string, error) {
	start := strings.Index(line, startStr)
	if start == -1 {
		return "", fmt.Errorf("find %s err, %s", startStr, line)
	}
	line = line[start+len(startStr):]
	end := strings.Index(line, endStr)
	if end == -1 {
		return "", fmt.Errorf("find %s err, %s", endStr, line)
	}
	line = line[:end]
	return line, nil
}

type PathQuery struct {
	T    string
	Resp model.PathQueryResponse
}

func parseLog(raw, t string) (PathQuery, error) {
	var resp model.PathQueryResponse
	raw = strings.ReplaceAll(raw, "\"\"", "\"")
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		log.Println(err, raw)
		return PathQuery{}, err
	}
	path := PathQuery{Resp: resp, T: t}
	return path, nil
}

func aggByDay(paths []PathQuery) {
	pathsByDay := map[string][]PathQuery{}
	for _, path := range paths {
		ss := strings.Split(path.T, " ")
		if _, ok := pathsByDay[ss[0]]; !ok {
			pathsByDay[ss[0]] = []PathQuery{path}
		} else {
			pathsByDay[ss[0]] = append(pathsByDay[ss[0]], path)
		}
	}

	for date, paths := range pathsByDay {
		log.Println(date, len(paths))
	}

	nodeMap := map[string]int{}
	for _, path := range pathsByDay["2024-05-31"] {
		resp := path.Resp
		log.Printf("%s %s\n", path.Resp.Sources[0].Node, path.Resp.Sources[1].Node)
		/*
			for _, item := range path.Resp.Sources {

			}
		*/
		nodeMap[resp.Sources[1].Node]++
	}
	log.Println("root node count", len(nodeMap))
	for node, cnt := range nodeMap {
		log.Println(node, cnt)
	}
	/*
		streamMap := map[string]int{}
		for _, path := range pathsByDay["2024-05-31"] {
			u := path.Resp.Sources[1].Url
			_, _, key, _ := util.ParseBucketKeyFromUrl(u)
			streamMap[key]++
		}
		for stream, cnt := range streamMap {
			log.Println(stream, cnt)
		}
	*/
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var file string
	flag.StringVar(&file, "f", "/Users/rigensen/Downloads/pathquery.csv", "文件")
	flag.Parse()

	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println("read fail", file, err)
		return
	}
	logs := []string{}
	lines := strings.Split(string(b), "\n")
	paths := []PathQuery{}
	for i, line := range lines {
		if i == 0 {
			continue
		}
		ss := splitIgnoringQuotes(line)
		if len(ss) < 18 {
			continue
		}
		t := ss[18][:23]
		raw, err := findStr(line, "resp:", " connectId=")
		if err != nil {
			//log.Println(err)
			continue
		}
		path, err := parseLog(raw, t)
		if err != nil {
			continue
		}
		paths = append(paths, path)
		logs = append(logs, raw)
	}
	log.Println(len(logs), len(paths))
	aggByDay(paths)
}

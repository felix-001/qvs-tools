package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

type M map[string]string

type Parser struct {
	Conf *Config
}

func NewParser(conf *Config) *Parser {
	return &Parser{Conf: conf}
}

type Keyword struct {
	Start string
	End   string
	Key   string
}

func (s *Parser) decodeErr() {
	re := fmt.Sprintf("grep \"15010400402000000000_15010400401320000656.*decode ps packet error\" * -R")
	logs, err := s.searchLogs("zz780", "qvs-rtp", re)
	if err != nil {
		log.Println(err)
		return
	}
	if len(logs) == 0 {
		log.Println("log empty")
		return
	}
	err = ioutil.WriteFile("decode.txt", []byte(logs), 0644)
	if err != nil {
		log.Println(err)
		return
	}
}

func (s *Parser) parseRtpLog(str string) (string, string, bool) {
	re := regexp.MustCompile(`\[(.*?)\]`)
	matches := re.FindAllStringSubmatch(str, -1)

	if len(matches) <= 0 {
		log.Println("not match", str)
		return "", "", false
	}
	if len(matches) < 3 {
		log.Println("match count not 4 real:", len(matches), matches)
		return "", "", false
	}
	return matches[0][1], matches[2][1], true
}

func (s *Parser) getCreateChLog(inviteTime, node string) (string, error) {
	// 2023-11-09 13:50:46.806 --> 2023-11-09 13:50:4
	start := time.Now()
	//inviteTime = strings.ReplaceAll(inviteTime, "/", "-")
	re := fmt.Sprintf("%s.*create_channel.*%s", inviteTime[:18], s.Conf.StreamId)
	data, err := s.searchLogs(node, "qvs-rtp", re)
	if err != nil {
		log.Println("search log err")
		return "", err
	}
	if data == "" {
		return "", fmt.Errorf("create ch log empty")
	}
	log.Println("get create ch log cost:", time.Since(start))
	return data, nil
}

func (s *Parser) getDeleteChLog(createChTime, node string) (string, error) {
	start := time.Now()
	re := fmt.Sprintf("delete_channel.*%s", s.Conf.StreamId)
	data, err := s.searchLogs(node, "qvs-rtp", re)
	if err != nil {
		log.Println("search log err")
		return "", err
	}
	if data == "" {
		return "", fmt.Errorf("delete ch log empty")
	}
	log.Println("get delete ch log cost:", time.Since(start))
	return s.getFirstLogAfterTimePoint(data, createChTime)
}

type StreamIdInfo struct {
	GbId  string
	ChId  string
	Start string
	End   string
}

func (s *Parser) getIds() (streamInfo StreamIdInfo) {
	if s.Conf.StreamId == "" {
		log.Fatal("check streamid err")
	}
	ss := strings.Split(s.Conf.StreamId, "_")
	streamInfo.GbId = ss[0]
	if len(ss) > 1 {
		streamInfo.ChId = ss[1]
	}
	if len(ss) > 2 {
		streamInfo.Start = ss[2]
		streamInfo.End = ss[3]
	}
	return
}

func (s *Parser) doSearch(node, service, query string, resultChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("fetching logs from", node)
	raw, err := s.searchLogs(node, service, query)
	if err != nil {
		log.Println("search log err", node, query, service)
		return
	}
	log.Println("fetch logs from", node, "done")
	resultChan <- raw
}

func (s *Parser) searchApiHandler(v interface{}) string {
	param := v.(TaskParam)
	service := strings.ToUpper(param.Service)
	result, err := s.searchApiLog(param.Node, service, param.Re)
	if err != nil {
		log.Println("search log err", param.Node, param.Service, param.Re)
		return ""
	}
	return result
}

func (s *Parser) searchQvsServerApiLog(re string) string {
	return s.fetchCenterLog("qvs-server", re, s.searchApiHandler)
}

func (s *Parser) RunParallelTask(params []interface{}, handler Handler) string {
	task := NewParaleelTask(params, handler)
	return task.Run()
}

type TaskParam struct {
	Node    string
	Re      string
	Service string
}

type TaskParamAllService struct {
	Node string
	Re   string
}

func (s *Parser) taskHandler(v interface{}) string {
	param := v.(TaskParam)
	result, err := s.searchLogs(param.Node, param.Service, param.Re)
	if err != nil {
		log.Println("search log err", param.Node, param.Service, param.Re)
		return ""
	}
	return result
}

func (s *Parser) taskHandlerAllService(v interface{}) string {
	param := v.(TaskParamAllService)
	result, err := s.searchLogsAllService(param.Node, param.Re)
	if err != nil {
		log.Println("search log err", param.Node, param.Re)
		return ""
	}
	return result
}

func (s *Parser) fetchCenterLog(service, re string, handler Handler) string {
	params := []interface{}{
		TaskParam{"jjh1445", re, service},
		TaskParam{"jjh250", re, service},
		TaskParam{"jjh1449", re, service},
		TaskParam{"bili-jjh9", re, service},
	}
	return s.RunParallelTask(params, handler)
}

func (s *Parser) fetchCenterAllServiceLogs(re string) string {
	params := []interface{}{
		TaskParamAllService{"jjh1445", re},
		TaskParamAllService{"jjh250", re},
		TaskParamAllService{"jjh1449", re},
		TaskParamAllService{"bili-jjh9", re},
	}
	return s.RunParallelTask(params, s.taskHandlerAllService)
}

func (s *Parser) fetchQvsServerLog(re string) string {
	return s.fetchCenterLog("qvs-server", re, s.taskHandler)
}

func (s *Parser) PullStreamLog() {
	re := fmt.Sprintf("start a  channel stream.*%s", s.Conf.StreamId)
	result := s.fetchQvsServerLog(re)
	log.Println("fetch pull stream log:", result)
}

func (s *Parser) isContain(v string, items []string) bool {
	for _, item := range items {
		if v == item {
			return true
		}
	}
	return false
}

func (s *Parser) getNodeByIP(ip string) (string, error) {
	nodes, err := s.getNodes()
	if err != nil {
		return "", err
	}
	for _, node := range nodes {
		if s.isContain(ip, node.Ips) {
			return node.ID, nil
		}
	}
	return "", fmt.Errorf("not found")
}

// 流断了，查询是哪里bye的
// 流量带宽异常，查询拉流的源是哪里: 按需拉流？按需截图？catalog重试？
// re := fmt.Sprintf("RTC play.*%s", s.Conf.StreamId)
// decode err
// 播放者的ip
// flv对端ip, "HttpFlvConnected" and "32050000491180000023_32050000491320000011"
func (s *Parser) Run() error {
	if s.Conf.StreamPullFail {
		s.streamPullFail()
		return nil
	}
	if s.Conf.Sip {
		start := time.Now()
		s.SearchSipLogs()
		log.Println("cost", time.Since(start))
		return nil
	}
	if s.Conf.PullStream {
		s.PullStreamLog()
		return nil
	}
	if s.Conf.HttpSrv {
		s.HttpSrvRun()
		return nil
	}
	result := s.fetchCenterAllServiceLogs(s.Conf.Re)
	log.Println(result)
	return nil
}

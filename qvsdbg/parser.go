package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type M map[string]string

type Parser struct {
	Conf  *Config
	nodes []Node
}

func NewParser(conf *Config) *Parser {
	return &Parser{Conf: conf}
}

type Keyword struct {
	Start string
	End   string
	Key   string
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
	re := fmt.Sprintf("%s.*create_channel.*%s", inviteTime[11:18], s.Conf.StreamId)
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

func (s *Parser) getNodeByIPWithCache(ip string) (string, error) {
	if s.nodes == nil {
		log.Println("get nodes")
		nodes, err := s.getNodes()
		if err != nil {
			return "", err
		}
		s.nodes = make([]Node, len(nodes))
		copy(s.nodes, nodes)
	}
	for _, node := range s.nodes {
		if s.isContain(ip, node.Ips) {
			return node.ID, nil
		}
	}
	return "", fmt.Errorf("not found")
}

type Pair struct {
	Key   string
	Value int
}

func (s *Parser) ParseNetstat() {
	b, err := ioutil.ReadFile(s.Conf.NetstatLogFile)
	if err != nil {
		log.Println("read fail", s.Conf.NetstatLogFile, err)
		return
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	unknow := 0
	private := 0
	statis := map[string]int{}
	localost := 0
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			log.Fatal("fields not 4")
		}
		//localAddress := fields[3]
		remoteAddress := fields[4]
		//fmt.Printf("Local Address: %s, Remote Address: %s\n", localAddress, remoteAddress)
		ss := strings.Split(remoteAddress, ":")
		if len(ss) != 2 {
			//log.Fatal("split ip:port err")
			log.Println("split ip:port err", remoteAddress)
		}
		ip := net.ParseIP(ss[0])
		if ip.IsPrivate() {
			private++
		}
		if ss[0] == "127.0.0.1" {
			localost++
		} else {
			node, err := s.getNodeByIPWithCache(ss[0])
			if err != nil {
				log.Println("unkonw ip:", ss[0])
				unknow++
			} else {
				log.Println("node:", node, "ip:", ss[0])
				statis[node]++
			}
		}
	}
	log.Println("unkonw:", unknow)
	log.Println("localhost:", localost)
	total := 0
	/*
		for k, v := range statis {
			log.Println(k, v)
			total += v
		}
	*/
	var pairs []Pair
	for key, value := range statis {
		pairs = append(pairs, Pair{key, value})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	for _, pair := range pairs {
		fmt.Printf("%s: %d\n", pair.Key, pair.Value)
		total += pair.Value
	}
	log.Println("total:", total)
	log.Println("private:", private)
}

type Fields struct {
	StreamId       string  `json:"streamId"`
	Status         string  `json:"status"`
	ReqId          string  `json:"reqId"`
	StartAt        int64   `json:"startAt"`
	Domain         string  `json:"domain"`
	BytesPerSecond string  `json:"bytesPerSecond"`
	Bytes          int     `json:"bytes"`
	Duration       string  `json:"duration"`
	Elapsed        int     `json:"elapsed"`
	VideoPerSecond string  `json:"videoPerSecond"`
	Video_fps      float64 `json:"video_fps"`
}

type Tags struct {
	Hub    string `json:"hub"`
	Uid    string `json:"uid"`
	Method string `json:"method"`
	Type   string `json:"type"`
	Domain string `json:"domain"`
}

type Point struct {
	Tags   Tags   `json:"tags"`
	Fields Fields `json:"fields"`
	Ts     int64  `json:"ts"`
}

func (s *Parser) ParsePoint() {
	b, err := ioutil.ReadFile(s.Conf.PointFile)
	if err != nil {
		log.Println("read fail", s.Conf.PointFile, err)
		return
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	streams := map[string]int{}
	internalSource := 0
	internalDestination := 0
	publish := 0
	play := 0
	disconnect := 0
	publishStream := map[string]int{}
	playStream := map[string]int{}
	interSrcStrms := map[string]int{}
	interDestStrms := map[string]int{}
	types := map[string]int{}
	bpsMap := map[string]int{}
	forwardMap := map[string]int{}
	uidMap := map[string]int{}

	for scanner.Scan() {
		line := scanner.Text()
		point := &Point{}
		if err := json.Unmarshal([]byte(line), point); err != nil {
			log.Println(err, line)
			continue
			//return
		}
		//log.Printf("%+v\n", point)
		streams[point.Fields.StreamId] = 1
		if point.Tags.Method == "internal source" {
			internalSource++
			interSrcStrms[point.Fields.StreamId]++
		}
		if point.Tags.Method == "internal destination" {
			internalDestination++
			interDestStrms[point.Fields.StreamId]++
		}
		if point.Tags.Method == "publish" {
			publish++
			publishStream[point.Fields.StreamId]++

		}
		if point.Tags.Method == "play" {
			play++
			playStream[point.Fields.StreamId]++
		}
		if point.Fields.Status == "disconnected" {
			disconnect++
		}
		types[point.Tags.Type]++

		if point.Fields.BytesPerSecond != "" {
			bytesPerSecond := []int{}
			raw := strings.ReplaceAll(point.Fields.BytesPerSecond, " ", ",")
			err := json.Unmarshal([]byte(raw), &bytesPerSecond)
			if err != nil {
				fmt.Println("Error:", err)
			}
			total := 0
			for _, sample := range bytesPerSecond {
				total += sample
			}
			bps := total / len(bytesPerSecond)
			if bps > bpsMap[point.Fields.StreamId] {
				bpsMap[point.Fields.StreamId] = bps
			}
		}

		if point.Tags.Method == "qvs forward" {
			forwardMap[point.Fields.StreamId]++
		}
		uidMap[point.Tags.Uid]++

	}
	//log.Println("total streams:", len(streams), "internal source:", internalSource, "internal destination:", internalDestination, "publish:", publish, "play:", play, "disconnect:", disconnect)
	for k, v := range types {
		log.Println(k, v)
	}
	/*
		for k, v := range bpsMap {
			log.Println(k, v/1000)
		}
	*/
	var pairs []Pair
	for key, value := range bpsMap {
		pairs = append(pairs, Pair{key, value})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	for _, pair := range pairs {
		fmt.Printf("%s: %d\n", pair.Key, pair.Value/1000)
	}
	log.Println("total streams:", len(streams))
	log.Println("inter soure streams:", len(interSrcStrms), "inter dest streams:", len(interDestStrms))
	log.Println("publish streams:", len(publishStream), "play streams:", len(playStream))
	log.Println("forward", len(forwardMap))

	i := 0
	for k, v := range interSrcStrms {
		log.Println(k, v)
		if i > 10 {
			break
		}
		i++
	}
	log.Println("uids :", len(uidMap))
	/*
		for k, v := range uidMap {
			log.Println(k, v)
		}
	*/
	var pairs1 []Pair
	for key, value := range uidMap {
		pairs1 = append(pairs1, Pair{key, value})
	}

	sort.Slice(pairs1, func(i, j int) bool {
		return pairs1[i].Value > pairs1[j].Value
	})

	for _, pair := range pairs1 {
		fmt.Printf("%s: %d\n", pair.Key, pair.Value)
	}
}

// 流断了，查询是哪里bye的
// 流量带宽异常，查询拉流的源是哪里: 按需拉流？按需截图？catalog重试？
// re := fmt.Sprintf("RTC play.*%s", s.Conf.StreamId)
// decode err, 15010400402000000000_15010400401320000656.*decode ps packet error
// 播放者的ip
// flv对端ip, "HttpFlvConnected" and "32050000491180000023_32050000491320000011"
func (s *Parser) Run() error {
	if s.Conf.NetstatLogFile != "" {
		s.ParseNetstat()
		return nil
	}
	if s.Conf.PointFile != "" {
		s.ParsePoint()
		return nil
	}
	if s.Conf.StreamPullFail {
		s.streamPullFail()
		return nil
	}
	if s.Conf.Sip {
		start := time.Now()
		result, err := s.GetSipMsgs(s.Conf.Re)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(result)
		log.Println("cost", time.Since(start))
		return nil
	}
	if s.Conf.PullStream {
		start := time.Now()
		re := fmt.Sprintf("start a  channel stream.*%s", s.Conf.StreamId)
		result := s.fetchCenterAllServiceLogs(re)
		log.Println(result)
		log.Println("cost:", time.Since(start))
		return nil
	}
	if s.Conf.HttpSrv {
		s.HttpSrvRun()
		return nil
	}
	if s.Conf.Node != "" {
		start := time.Now()
		result, err := s.searchLogs(s.Conf.Node, "qvs-rtp", s.Conf.Re)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(result)
		log.Println("cost:", time.Since(start))
		return nil
	}
	if s.Conf.Bye {
		if s.Conf.StreamId == "" {
			log.Println("need streamid")
			return nil
		}
		streamInfo := s.getIds()
		start := time.Now()
		// 收到catalog拉流，autostrat
		re := fmt.Sprintf("rebuild strean.*%s.*%s|", streamInfo.GbId, streamInfo.ChId)
		// 按需拉流
		re += fmt.Sprintf("start a.*stream.*%s|", s.Conf.StreamId)
		// 停止拉流
		re += fmt.Sprintf("devices/%s/stop.*%s|", streamInfo.GbId, streamInfo.ChId)
		// 按需截图
		re += fmt.Sprintf("streams/%s/snap|", s.Conf.StreamId)
		// 一分钟无人观看关闭
		re += fmt.Sprintf("CloseStream.*%s", s.Conf.StreamId)
		result := s.fetchCenterAllServiceLogs(re)
		log.Println(result)
		rtpNodeId, err := s.getInviteRtpNode(streamInfo.GbId, streamInfo.ChId)
		if err != nil {
			log.Fatalln(err)
		}
		// 连接被对端断开
		re = fmt.Sprintf("%s.*reset by perr", s.Conf.StreamId)
		rtpRes, err := s.searchLogs(rtpNodeId, "qvs-rtp", re)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(rtpRes)
		log.Println("cost:", time.Since(start))
		return nil
	}
	start := time.Now()
	result := s.fetchCenterAllServiceLogs(s.Conf.Re)
	log.Println(result)
	log.Println("cost:", time.Since(start))
	return nil
}

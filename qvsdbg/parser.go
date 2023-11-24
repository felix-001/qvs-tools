package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type M map[string]string

type Parser struct {
	Conf *Config
	pdr  *Pdr
}

func NewParser(conf *Config) *Parser {
	pdr := NewPdr(conf)
	return &Parser{Conf: conf, pdr: pdr}
}

func (s *Parser) getValue(line, start, end string) (string, bool) {
	reg := fmt.Sprintf("%s(.*?)%s", start, end)
	re := regexp.MustCompile(reg)
	matchs := re.FindStringSubmatch(line)
	if len(matchs) < 1 {
		return "", false
	}
	return strings.TrimSpace(matchs[1]), true
}

type Keyword struct {
	Start string
	End   string
	Key   string
}

func (s *Parser) uniq(data string) M {
	ss := strings.Split(data, "\n")
	m := M{}
	for _, s1 := range ss {
		streamid, match := s.getValue(s1, "2xenzw32d1rf9/", ", api")
		if !match {
			log.Printf("not match, %s\n", s1)
			continue
		}
		if m[streamid] != "" {
			continue
		}
		m[streamid] = s1
	}
	return m
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
	if len(matches) < 4 {
		log.Println("match count not 4 real:", len(matches), matches)
		return "", "", false
	}
	return matches[0][1], matches[3][1], true
}

func (s *Parser) filterLogByDate(in, start, end string) ([]string, error) {
	ss := strings.Split(in, "\n")
	res := []string{}
	for _, str := range ss {
		if strings.Contains(str, "Pseudo-terminal") {
			continue
		}
		if str == "" {
			continue
		}
		time, _, match := s.parseRtpLog(str)
		if !match {
			continue
		}
		if time > start {
			if end == "" {
				res = append(res, str)
				continue
			}
			if time < end {
				res = append(res, str)
			}
		}
	}
	return res, nil
}

func (s *Parser) filterLogByTask(ss []string) map[string][]string {
	m := map[string][]string{}
	for _, str := range ss {
		if str == "" {
			continue
		}
		_, task, match := s.parseRtpLog(str)
		if !match {
			continue
		}
		if _, ok := m[task]; !ok {
			m[task] = []string{str}
			continue
		}
		m[task] = append(m[task], str)
	}
	return m
}

func insertString(original, insert string, pos int) string {
	if pos < 0 || pos > len(original) {
		return original
	}

	return original[:pos] + insert + original[pos:]
}

func (s *Parser) getCreateChLog(inviteTime, node string) (string, error) {
	// 2023-11-09 13:50:46.806 --> 2023-11-09 13:50:4
	start := time.Now()
	inviteTime = strings.ReplaceAll(inviteTime, "/", "-")
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

func (s *Parser) getStreamId(callId, ssrc string) (string, error) {
	du := 4 * 24 * time.Hour
	end := time.Now().UnixMilli()
	start := end - du.Milliseconds()
	re := fmt.Sprintf("repo = \"logs\" and \"%s\" and \"%s*\"", ssrc, callId)
	resp, err := s.pdr.FetchLog(re, start, end)
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println(resp)
	gbid, match := s.getValue(resp.Rows[0].Raw.Value, "gbId:", "chId:")
	if !match {
		log.Println("get gbid err")
		return "", fmt.Errorf("get gbid err")
	}
	log.Println("gbid:", gbid)
	chid, match := s.getValue(resp.Rows[0].Raw.Value, "chId:", "resp")
	if !match {
		log.Println("get chid err")
		return "", fmt.Errorf("get chid err")
	}
	log.Println("chid:", chid)
	return fmt.Sprintf("%s_%s", gbid, chid), nil
}

func (s Parser) getInviteMsg2(date, ssrc string) (string, string, error) {
	du := 4 * 24 * time.Hour
	end := time.Now().UnixMilli()
	start := end - du.Milliseconds()
	re := fmt.Sprintf("repo = \"sip_msg_dump2\" and \"0%s\" and \"%s*\"", ssrc, date[:10])
	log.Println(re)
	resp, err := s.pdr.FetchLog(re, start, end)
	if err != nil {
		log.Println(err)
		return "", "", err
	}
	//log.Printf("%#v\n", resp)
	log.Println(resp.Rows[0].Raw.Value)
	log.Println(resp.Rows[1].Raw.Value)
	inviteTime, _, match := s.parseRtpLog(resp.Rows[0].Raw.Value)
	if !match {
		log.Println("get invite time err")
		return "", "", fmt.Errorf("get invite time err")
	}

	pos := strings.Index(resp.Rows[0].Raw.Value, "send_message:")
	if pos == -1 {
		log.Println("find send_message: err")
		return "", "", err
	}
	raw := resp.Rows[0].Raw.Value[pos+len("send_message:")+1:]
	idx := strings.Index(raw, "Content-Length")
	if idx == -1 {
		log.Println("find Content-Length err")
		return "", "", err
	}
	idx2 := strings.Index(raw[idx:], "\r\n")
	if idx2 == -1 {
		log.Println("find idx2 err")
		return "", "", err
	}
	idx += idx2 + 2
	raw = insertString(raw, "\r\n", idx)
	return inviteTime, raw, nil

}

func (s *Parser) getSsrc(str string) (string, error) {
	val, match := s.getValue(str, "sts=", ", muxer")
	if !match {
		return "", fmt.Errorf("get sts err")
	}
	ss := strings.Split(val, "/")
	if len(ss) != 3 {
		return "", fmt.Errorf("ss not 3")
	}
	num, err := strconv.ParseInt(ss[2][2:], 16, 64)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", num), nil
}

type SSRCInfo struct {
	SSRC string
	Date string
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

func (s *Parser) getValByRegex(str, re string) (string, error) {
	regex := regexp.MustCompile(re)
	matchs := regex.FindStringSubmatch(str)
	if len(matchs) < 1 {
		return "", fmt.Errorf("not match, str: %s re: %s", str, re)
	}
	return matchs[1], nil
}

func (s *Parser) getNewestLog(logs string) (string, error) {
	ss := strings.Split(logs, "\r\n")
	if len(ss) == 1 {
		return logs, nil
	}
	newestLog := ""
	newestDateTime := ""
	for _, str := range ss {
		if str == "" {
			continue
		}
		if strings.Contains(str, "Pseudo-terminal") {
			continue
		}
		dateTime, err := s.getValByRegex(str, `(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}.\d+)`)
		if err != nil {
			return "", err
		}
		if newestLog == "" {
			newestLog = str
			newestDateTime = dateTime
			continue
		}
		if dateTime > newestDateTime {
			newestLog = str
			newestDateTime = dateTime
		}
	}
	if newestLog == "" {
		return "", fmt.Errorf("no valid log found")
	}
	return newestLog, nil
}

func (s *Parser) query(Keywords []string) (query string) {
	for i, keyword := range Keywords {
		if i < len(Keywords)-1 {
			query += fmt.Sprintf("%s.*", keyword)
		} else {
			query += keyword
		}
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

func (s *Parser) multiLineQuery(Keywords []string) (query string) {
	//query = "(?s)(?<=<--------------------------------------------------------------------------------------------------->).*?"
	query = "(?s)(---).*?"
	for _, keyword := range Keywords {
		query += keyword + ".*?"
	}
	query += "(---)"
	return
}

func (s *Parser) searchLog() {
	if s.Conf.Node == "" || s.Conf.Service == "" {
		flag.PrintDefaults()
		log.Fatalln("check param err")
	}
	if s.Conf.Node == "center" {
		nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
		out := ""
		for _, node := range nodes {
			result, err := s.searchLogs(node, s.Conf.Service, s.Conf.Re)
			if err != nil {
				log.Fatalln(err)
			}
			out += result
		}
		log.Println(out)
		return
	}
	result, err := s.searchLogs(s.Conf.Node, s.Conf.Service, s.Conf.Re)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(result)

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

func (s *Parser) searchApiLogs() {
	if s.Conf.Service == "" {
		flag.PrintDefaults()
		log.Fatalln("need service")
	}
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	out := ""
	service := strings.ToUpper(s.Conf.Service)
	for _, node := range nodes {
		result, err := s.searchApiLog(node, service, s.Conf.Re)
		if err != nil {
			log.Fatalln(err)
		}
		out += result
	}
	log.Println(out)
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

func (s *Parser) taskHandler(v interface{}) string {
	param := v.(TaskParam)
	result, err := s.searchLogs(param.Node, param.Service, param.Re)
	if err != nil {
		log.Println("search log err", param.Node, param.Service, param.Re)
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

func (s *Parser) fetchQvsServerLog(re string) string {
	return s.fetchCenterLog("qvs-server", re, s.taskHandler)
}

func (s *Parser) PullStreamLog() {
	re := fmt.Sprintf("start a  channel stream.*%s", s.Conf.StreamId)
	result := s.fetchQvsServerLog(re)
	log.Println("fetch pull stream log:", result)
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
	if s.Conf.Api {
		if s.Conf.Service == "qvs-server" {
			result := s.searchQvsServerApiLog(s.Conf.Re)
			log.Println(result)
			return nil
		}
		s.searchApiLogs()
		return nil
	}
	if s.Conf.Re != "" {
		s.searchLog()
		return nil
	}
	if len(s.Conf.Keywords) > 0 {
		s.SearchSipLogs()
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
	//log.Println(getAllSipRawFiles2())
	msgs, _ := GetSipMsgs("202076923212,4491783")
	log.Println(msgs)
	return nil
}

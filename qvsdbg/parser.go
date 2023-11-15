package main

import (
	"bufio"
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

func (s *Parser) adminGet(path string) (string, error) {
	uri := fmt.Sprintf("http://%s:7277/v1/%s", s.Conf.AdminAddr, path)
	headers := M{"authorization": "QiniuStub uid=0"}
	return httpReq("GET", uri, "", headers)
}

func (s *Parser) adminPost(path, body string) (string, error) {
	uri := fmt.Sprintf("http://%s:7277/v1/%s", s.Conf.AdminAddr, path)
	headers := map[string]string{
		"authorization": "QiniuStub uid=0",
		"Content-Type":  "application/json",
	}
	return httpReq("POST", uri, body, headers)
}

func (s *Parser) searchLogs(node, service, re string) (string, error) {
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/%s/_package/run;grep -E -h '%s' * -R\\\"\"", node, service, re)
	if s.Conf.Verbose {
		log.Println(cmd)
	}
	return RunCmd(cmd)
}

// 遇到一个匹配的就停止
func (s *Parser) searchLogsOne(node, service, re string) (string, error) {
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/%s/_package/run;grep -E -h -m 1 '%s' * -R \\\"\"", node, service, re)
	if s.Conf.Verbose {
		log.Println(cmd)
	}
	return RunCmd(cmd)
}

func (s *Parser) searchLogsMultiLine(node, service, re string) (string, error) {
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/%s/_package/run/auditlog/sip_dump;grep -h -Pzo '%s' * -R\\\"\"", node, service, re)
	log.Println(cmd)
	return RunCmd(cmd)
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

type Rule struct {
	Match    string
	Keywords []Keyword
}

var rules = []Rule{
	{
		Match: "invite ok",
		Keywords: []Keyword{
			{
				Start: "rtpAccessIp:",
				End:   "callId",
				Key:   "rtpIp",
			},
			{
				Start: "callId:",
				End:   "____",
				Key:   "callId",
			},
			{
				Start: "ssrc:",
				End:   "host",
				Key:   "ssrc",
			},
			{
				Start: "rtpPort:",
				End:   "$",
				Key:   "rtpPort",
			},
		},
	},
}

func (s *Parser) parseInviteBye(data string) error {
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		result := M{}
		for _, rule := range rules {
			re := regexp.MustCompile(rule.Match)
			if re.MatchString(line) {
				for _, keyword := range rule.Keywords {
					val, match := s.getValue(line, keyword.Start, keyword.End)
					if match {
						result[keyword.Key] = val
					}
				}
			}
		}
		for k, v := range result {
			log.Println(k, v)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
	}
	return nil
}

func (s *Parser) inviteBye() error {
	data := ""
	//nodes := []string{"jjh1445", "jjh1449", "jjh250", "bili-jjh9"}
	ss := strings.Split(s.Conf.StreamId, "_")
	nodes := []string{"jjh1445"}
	for _, node := range nodes {
		invite := fmt.Sprintf("invite ok.*%s", ss[0])
		bye := fmt.Sprintf("bye ok.*%s", ss[1])
		re := fmt.Sprintf("%s|%s", invite, bye)
		res, err := s.searchLogs(node, "qvs-server", re)
		if err != nil {
			log.Println(res, err)
			return err
		}
		//log.Println(res)
		data += res
	}
	if err := s.parseInviteBye(data); err != nil {
		log.Println(err)
	}
	err := ioutil.WriteFile("test.txt", []byte(data), 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (s *Parser) getZZList() (string, error) {
	cmd := fmt.Sprint("ssh -t liyuanquan@10.20.34.27 \"floy version qvs-rtp 2>/dev/null | grep zz  | awk '{print $1}'\"")
	log.Println(cmd)
	return RunCmd(cmd)
}

func (s *Parser) rtcLog() error {
	s1, err := s.getZZList()
	if err != nil {
		log.Println(err)
		return err
	}
	//log.Println(s1)
	ss := strings.Split(s1, "\n")
	nodes := []string{}
	for _, s2 := range ss {
		ss1 := strings.Split(s2, ",")
		nodes = append(nodes, ss1[0])
	}
	log.Println(nodes)

	data := ""
	for _, node := range nodes[1:] {
		if node == " " || node == "" {
			continue
		}
		re := fmt.Sprintf("RTC play.*%s", s.Conf.StreamId)
		res, err := s.searchLogs(node, "qvs-rtp", re)
		if err != nil {
			log.Println(err)
			continue
			//return err
		}
		data += res
		log.Println("len:", len(data))
	}
	err = ioutil.WriteFile("out.txt", []byte(data), 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
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

func (s *Parser) calc() {
	b, err := ioutil.ReadFile("./11-03.txt")
	if err != nil {
		log.Println("read fail", "./11-03.txt", err)
		return
	}
	ss := strings.Split(string(b), "\n")
	m := map[string]int{}
	for _, s1 := range ss {
		streamid, match := s.getValue(s1, "2xenzw32d1rf9/", ", api")
		if !match {
			log.Printf("not match, %s\n", s1)
			continue
		}
		_, ok := m[streamid]
		if ok {
			m[streamid] += 1
		} else {
			m[streamid] = 1
		}
	}
	log.Println("len m:", len(m))
	for k, v := range m {
		log.Printf("%s : %d\n", k, v)
	}
}

func (s *Parser) parseRtcLog() {
	b, err := ioutil.ReadFile("./11-03.txt")
	if err != nil {
		log.Println("read fail", "./11-03.txt", err)
		return
	}
	m := s.uniq(string(b))
	log.Println("m len:", len(m))
	/*
		for i := 0; i < 24; i++ {
			s1 := fmt.Sprintf("2023-11-03 %02d", i)
			cmd := fmt.Sprintf("grep \"%s\" /Users/liyuanquan/workspace/qvs-tools/qvsdbg/11-03.txt", s1)
			//log.Println(cmd)
			res, err := RunCmd(cmd)
			if err != nil {
				log.Println(err, res)
				continue
			}
			m := s.uniq(res)
			log.Printf("%02d : %d\n", i, len(m))

			if i == 17 {
				//log.Printf("%#v\n", m)
				ss := []string{}
				for k, _ := range m {
					//log.Println(k)
					ss = append(ss, k)
				}
				sort.Strings(ss)
				log.Println(ss)
			}
		}
	*/

	/*
		for i := 0; i < 60; i++ {
			s := fmt.Sprintf("2023-11-03 17:%02d", i)
			cmd := fmt.Sprintf("grep \"%s\" /Users/liyuanquan/workspace/qvs-tools/qvsdbg/11-03.txt", s)
			//log.Println(cmd)
			res, err := RunCmd(cmd)
			if err != nil {
				log.Println(err, res)
				continue
			}
			ss := strings.Split(res, "\n")
			log.Printf("%02d : %d\n", i, len(ss))
		}
	*/

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

func (s *Parser) rtpNoMuxer() (string, error) {
	s1, err := s.getZZList()
	if err != nil {
		log.Println(err)
		return "", err
	}
	//log.Println(s1)
	ss := strings.Split(s1, "\n")
	nodes := []string{}
	for _, s2 := range ss {
		ss1 := strings.Split(s2, ",")
		nodes = append(nodes, ss1[0])
	}
	log.Println(nodes)
	// nodes := []string{"zz780"}
	data := ""
	for i, node := range nodes {
		if node == "" {
			continue
		}
		re := fmt.Sprint("udp gb28181 rtp enqueue.*111.56.244.163.*nil")
		logs, err := s.searchLogs(node, "qvs-rtp", re)
		if err != nil {
			log.Println(err)
			continue
		}
		/*
			err = ioutil.WriteFile("./zz/"+node+".txt", []byte(logs), 0644)
			if err != nil {
				log.Println(err)
				return
			}
		*/
		log.Printf("%d -> %d\n", i, len(nodes))
		data += logs
	}
	err = ioutil.WriteFile("rtpNoMuxer.txt", []byte(data), 0644)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return data, nil
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

/*
func (s *Parser) logSortByTime(ss []string) []string {
	for _, str := range ss {
		time, _, match := s.parseRtpLog(str)
		if !match {
			continue
		}
	}
}
*/

func insertString(original, insert string, pos int) string {
	if pos < 0 || pos > len(original) {
		return original
	}

	return original[:pos] + insert + original[pos:]
}

func (s *Parser) getRtpNodeByPdr(ip string) (string, error) {
	du := 15 * time.Minute
	end := time.Now().UnixMilli()
	start := end - du.Milliseconds()
	query := fmt.Sprintf("repo = \"logs\" and \"%s\" and \"/stream/publish/check\"", ip)
	pdrLog, err := s.pdr.FetchLog(query, start, end)
	if err != nil {
		log.Println(err)
		return "", err
	}
	//log.Println(pdrLog.Total, pdrLog.Rows[0].Raw.Value)
	nodeId, match := s.getValue(pdrLog.Rows[0].Raw.Value, "\"nodeId\":\"", "\",")
	if !match {
		return "", fmt.Errorf("parse node Id err")
	}
	return nodeId, nil
}

func (s *Parser) getRtpIp() (string, error) {
	du := 3 * 24 * time.Hour
	end := time.Now().UnixMilli()
	start := end - du.Milliseconds()
	resp, err := s.pdr.FetchLog("repo = \"logs\" and \"41468169\" and \"invite ok\"", start, end)
	if err != nil {
		log.Println(err)
		return "", err
	}
	//log.Println(resp.Rows[0].Raw.Value)
	rtpIp, match := s.getValue(resp.Rows[0].Raw.Value, "rtpAccessIp:", "callId")
	if !match {
		return "", fmt.Errorf("parse rtp ip err")
	}
	return rtpIp, nil
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

func (s *Parser) getByeMsg(callId string) (string, error) {
	du := 4 * 24 * time.Hour
	end := time.Now().UnixMilli()
	start := end - du.Milliseconds()
	query := fmt.Sprintf("repo = \"sip_msg_dump2\" and \"bye\" and \"%s\"", callId)
	pdrLog, err := s.pdr.FetchLog(query, start, end)
	if err != nil {
		log.Println(err)
		return "", err
	}
	//log.Println(pdrLog.Total, pdrLog.Rows[0].Raw.Value)
	//log.Println(pdrLog.Total, pdrLog.Rows[1].Raw.Value)
	return pdrLog.Rows[0].Raw.Value, nil
}

func (s *Parser) rtpNoMuxerAllLog(date, ssrc string) {
	inviteTime, inviteMsg, err := s.getInviteMsg2(date, ssrc)
	if err != nil {
		log.Println("err")
		return
	}
	log.Println("invite time:", inviteTime)
	callId, err := GetCallId(inviteMsg)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("callid", callId)
	byeMsg, err := s.getByeMsg(callId)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(byeMsg)
	rtpIp, err := s.getRtpIp()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("rtpIp:", rtpIp)
	rtpNode, err := s.getRtpNodeByPdr(rtpIp)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("rtpNode:", rtpNode)
	streamid, err := s.getStreamId(callId, "41468169")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("streamid:", streamid)
	createChLog, err := s.getCreateChLog(inviteTime, "zz450")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(createChLog)
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

func (s *Parser) getSsrcList(ss []string) ([]SSRCInfo, error) {
	m := map[string]int{}
	for _, str := range ss {
		dateTime, _, match := s.parseRtpLog(str)
		if !match {
			log.Println("parse rtp log err", str)
			continue
		}
		date := dateTime[:10]
		ssrc, err := s.getSsrc(str)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		key := fmt.Sprintf("%s_%s", date, ssrc)
		if _, ok := m[key]; !ok {
			m[key] = 0
			continue
		}
		m[key]++
	}
	ssrcs := []SSRCInfo{}
	for k, v := range m {
		if v >= 10 {
			//log.Println(k, v)
			ss := strings.Split(k, "_")
			ssrcs = append(ssrcs, SSRCInfo{SSRC: ss[1], Date: ss[0]})
		}
	}
	return ssrcs, nil
}

func (s *Parser) rtpNoMuxerLog() error {
	b, err := ioutil.ReadFile("rtpNoMuxer.txt")
	if err != nil {
		log.Println("read fail", "rtpNoMuxer.txt", err)
		return err
	}
	data := string(b)
	str := strings.ReplaceAll(data, "0m[", "")
	ss, err := s.filterLogByDate(str, "2023-11-05 00:00:00", "2023-11-09 00:00:00")
	if err != nil {
		log.Println(err)
		return err
	}
	ssrcs, err := s.getSsrcList(ss)
	if err != nil {
		return err
	}
	log.Println(ssrcs)

	for i, ssrc := range ssrcs {
		if i < 2 {
			s.rtpNoMuxerAllLog(ssrc.Date, ssrc.SSRC)
		}
	}
	return nil
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

func (s *Parser) getInviteLog() (string, error) {
	streamInfo := s.getIds()
	keywords := []string{"invite ok", streamInfo.GbId}
	if streamInfo.ChId != "" {
		keywords = append(keywords, streamInfo.ChId)
	}
	query := s.query(keywords)
	resultChan := make(chan string)
	wg := sync.WaitGroup{}
	wg.Add(4)
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	for _, node := range nodes {
		go s.doSearch(node, "qvs-server", query, resultChan, &wg)
	}
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	var finalResult string
	for str := range resultChan {
		finalResult += str
	}
	logs, err := s.getNewestLog(finalResult)
	if err != nil {
		log.Println("get newest loggg err:", err)
		return "", err
	}
	//log.Println(node, raw)
	return logs, nil
}

type InvitInfo struct {
	CallId  string
	SSRC    string
	RtpIp   string
	SipNode string
	RtpPort string
	Time    string
}

func (s *Parser) getInviteInfo() (inviteInfo InvitInfo, err error) {
	raw, err := s.getInviteLog()
	if err != nil {
		return
	}
	inviteInfo.CallId, err = s.getValByRegex(raw, `callId: (\d+)`)
	if err != nil {
		return
	}
	inviteInfo.SSRC, err = s.getValByRegex(raw, `ssrc:  (\d+)`)
	if err != nil {
		return
	}
	inviteInfo.RtpIp, err = s.getValByRegex(raw, `rtpAccessIp: (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	if err != nil {
		return
	}
	inviteInfo.SipNode, err = s.getValByRegex(raw, `rtpNode: (\S+)`)
	if err != nil {
		return
	}
	inviteInfo.RtpPort, err = s.getValByRegex(raw, `rtpPort: (\d+)`)
	if err != nil {
		return
	}
	inviteInfo.Time, err = s.getValByRegex(raw, `(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}.\d+)`)
	if err != nil {
		return
	}
	return
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

func (s *Parser) getInviteMsg(inviteInfo *InvitInfo, chid string) (string, error) {
	//streamInfo := s.getIds()
	//Keywords := []string{chid, inviteInfo.CallId, inviteInfo.RtpIp, "0" + inviteInfo.SSRC}
	Keywords := []string{chid, inviteInfo.CallId, "0" + inviteInfo.SSRC}
	query := s.multiLineQuery(Keywords)
	return s.searchLogsMultiLine(inviteInfo.SipNode, "qvs-sip", query)
}

func (s *Parser) getChidOfIPC(node, gbid string) (string, error) {
	Keywords := []string{"real chid of", gbid}
	query := s.query(Keywords)
	raw, err := s.searchLogsOne(node, "qvs-sip", query)
	if err != nil {
		return "", err
	}
	chid, err := s.getValByRegex(raw, `real chid of \d+ is (\d+)`)
	if err != nil {
		return "", err
	}
	//log.Println("chid:", chid)
	return chid, nil
}

func (s *Parser) getCreateChLogs(inviteTime, streamId, nodeId string) (string, error) {
	re := fmt.Sprintf("%s.*create_channel.*%s", inviteTime[:18], streamId)
	return s.searchLogs(nodeId, "qvs-rtp", re)
}

func (s *Parser) getRtpNode(rtpIp string) (string, error) {
	start := time.Now()
	re := fmt.Sprintf("/stream/publish/check.*%s", rtpIp)
	result, err := searchThemisd(re)
	if err != nil {
		return "", err
	}
	log.Println("get rtp node cost:", time.Since(start))
	return s.getValByRegex(result, `"nodeId":"(.*?)"`)
}

func (s *Parser) getRtpLog(taskId, nodeId string) (string, error) {
	re := fmt.Sprintf("%s.*got first|%s.*delete_channel|%s.*stream idle timeout", taskId, taskId, taskId)
	return s.searchLogs(nodeId, "qvs-rtp", re)
}

/*
 * invite √
 * invite resp √
 * bye √
 * bye resp √
 * create channel √
 * delete channel
 * channel remove
 * got first
 * tcp attach
 * inner stean
 * catalog invite
 */
func (s *Parser) streamPullFail() {
	streamInfo := s.getIds()
	inviteInfo, err := s.getInviteInfo()
	if err != nil {
		log.Println("get invite info err:", err)
		return
	}
	log.Println("callid:", inviteInfo.CallId, "ssrc:", inviteInfo.SSRC, "rtpIp:", inviteInfo.RtpIp, "sipNode:", inviteInfo.SipNode, "rtpPort:", inviteInfo.RtpPort, "time:", inviteInfo.Time)
	if streamInfo.ChId == "" {
		chid, err := s.getChidOfIPC(inviteInfo.SipNode, streamInfo.GbId)
		if err != nil {
			log.Fatalln(err)
		}
		streamInfo.ChId = chid
	}
	log.Println("real chid:", streamInfo.ChId)
	start := time.Now()
	params := streamInfo.ChId + "," + inviteInfo.CallId
	sipMsgs, err := GetSipMsg(params)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println(sipMsgs)
	log.Println("cost:", time.Since(start))
	msgs := s.splitSipMsg(sipMsgs)
	log.Println("msgs: ", len(msgs))
	//log.Println(msgs)
	rtpNodeId, err := s.getRtpNode(inviteInfo.RtpIp)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("rtpNodeId:", rtpNodeId)
	createChLog, err := s.getCreateChLog(inviteInfo.Time, rtpNodeId)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("createChLog:", createChLog)
	_, taskId, match := s.parseRtpLog(createChLog)
	if !match {
		log.Fatalln("get task id from create ch log err")
	}
	log.Println("taskId:", taskId)
	rtpLog, err := s.getRtpLog(taskId, rtpNodeId)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("rtpLog:", rtpLog)
}

func (s *Parser) splitSipMsg(raw string) []string {
	ss := strings.Split(raw, "\r\n")
	sipMsg := ""
	msgs := []string{}
	for _, str := range ss {
		if strings.Contains(str, "---") {
			if sipMsg == "" {
				continue
			}
			msgs = append(msgs, sipMsg)
			sipMsg = ""
			continue
		}
		sipMsg += str + "\r\n"
		if strings.Contains(str, "Content-Length") {
			sipMsg += "\r\n"
		}
	}
	return msgs
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

func (s *Parser) Run() error {
	if s.Conf.StreamPullFail {
		s.streamPullFail()
	}
	if s.Conf.Re != "" {
		s.searchLog()
	}
	return nil
}

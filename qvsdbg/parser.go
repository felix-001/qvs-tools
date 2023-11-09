package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
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
	//log.Println(cmd)
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
	nodes := []string{"jjh1445"}
	for _, node := range nodes {
		invite := fmt.Sprintf("invite ok.*%s", s.Conf.GbId)
		bye := fmt.Sprintf("bye ok.*%s", s.Conf.GbId)
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
		re := fmt.Sprintf("RTC play.*%s", s.Conf.GbId)
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

func (s *Parser) getRtpNode(ip string) (string, error) {
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

func (s *Parser) getCreateChLog(inviteTime, streamid string) (string, error) {
	// 2023-11-09 13:50:46.806 --> 2023-11-09 13:50:4
	re := fmt.Sprintf("%s.*create_channel.*%s", inviteTime[:18], streamid)
	data, err := s.searchLogs("zz450", "qvs-rtp", re)
	if err != nil {
		log.Println("search log err")
		return "", err
	}
	log.Println(data)
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

func (s Parser) getInviteMsg(date, ssrc string) (string, string, error) {
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
	inviteTime, inviteMsg, err := s.getInviteMsg(date, ssrc)
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
	rtpNode, err := s.getRtpNode(rtpIp)
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
	createChLog, err := s.getCreateChLog(inviteTime, streamid)
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

// rtp日志找到muxer为nil的ssrc
// invite和resp信令
// 根据callid找到bye
// ssrc和streamid找到create channel和delete channel日志
func (s *Parser) Run() error {
	//s.rtpNoMuxerAllLog()
	//s.inviteBye()
	//s.parseRtcLog()
	//s.decodeErr()
	//s.calc()
	/*
		data, err := s.rtpNoMuxer()
		if err != nil {
			log.Println(err)
			return err
		}
	*/
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

// 捞取日志集合, 各种and集合

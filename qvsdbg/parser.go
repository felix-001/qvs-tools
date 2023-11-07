package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
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

func (s *Parser) rtpNoMuxer() {
	s1, err := s.getZZList()
	if err != nil {
		log.Println(err)
		return
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
	for _, node := range nodes {
		if node == "" {
			continue
		}
		re := fmt.Sprint("udp gb28181 rtp enqueue.*111.56.244.163.*nil")
		logs, err := s.searchLogs(node, "qvs-rtp", re)
		if err != nil {
			log.Println(err)
			continue
		}
		data += logs
	}
	err = ioutil.WriteFile("rtpNoMuxer.txt", []byte(data), 0644)
	if err != nil {
		log.Println(err)
		return
	}
}

func (s *Parser) Run() error {
	//s.inviteBye()
	//s.parseRtcLog()
	//s.decodeErr()
	//s.calc()
	//s.rtpNoMuxer()
	resp, err := s.pdr.FetchLog("repo = \"logs\" and \"invite\"", time.Now().Unix()-600, time.Now().Unix())
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(resp)
	return nil
}

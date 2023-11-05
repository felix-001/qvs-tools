package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

type M map[string]string

type Parser struct {
	Conf *Config
}

func NewParser(conf *Config) *Parser {
	return &Parser{Conf: conf}
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

type KeyWord struct {
	Start string
	End   string
	Key   string
}

type Result struct {
	Key   string
	Value string
}

func re(start, end string) string {
	return fmt.Sprintf("%s(.*)%s", start, end)
}

func (s *Parser) parseLine(line string, keywords []KeyWord) ([]Result, error) {
	regex := re(keywords[0].Start, keywords[1].End)
	for i := 1; i < len(keywords); i++ {
		regex += ".*?" + re(keywords[i].Start, keywords[i].End)
	}
	re := regexp.MustCompile(regex)
	matchs := re.FindStringSubmatch(line)
	if len(matchs) < len(keywords) {
		return nil, fmt.Errorf("parse %s err", line)
	}
	results := []Result{}
	for i, match := range matchs {
		results = append(results, Result{Key: keywords[i].Key, Value: match})
	}
	return results, nil
}

func (s *Parser) parseInviteBye(data string) error {
	log.Println(data)
	//scanner := bufio.NewScanner(strings.NewReader(data))
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		//line := scanner.Text()
		log.Println(line)
		keywords := []KeyWord{
			{
				Key:   "rtpIp",
				Start: "rtpAccessIp",
				End:   "callId",
			},
			{
				Key:   "callId",
				Start: "callId",
				End:   "___",
			},
			{
				Key:   "ssrc",
				Start: "ssrc:",
				End:   "host",
			},
			{
				Key:   "rtpPort",
				Start: "rtpPort:",
				End:   "$",
			},
		}
		results, err := s.parseLine(line, keywords)
		if err != nil {
			//log.Println(err)
			return err
		}
		log.Println(results)
	}
	/*
		if err := scanner.Err(); err != nil {
			fmt.Println("Error:", err)
		}
	*/
	return nil
}

func (s *Parser) inviteBye() error {
	data := ""
	nodes := []string{"jjh1445", "jjh1449", "jjh250", "bili-jjh9"}
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
	return nil
}

func (s *Parser) Run() error {
	s.inviteBye()
	return nil
}

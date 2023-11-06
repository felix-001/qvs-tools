package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
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
		/*
			rtpIp, match := s.getValue(line, "rtpAccessIp:", "callId")
			if match {
				log.Println("rtpIp:", rtpIp)
			}
			callid, match := s.getValue(line, "callId:", "____")
			if match {
				log.Println("callid:", callid)
			}
			ssrc, match := s.getValue(line, "ssrc:", "host")
			if match {
				log.Println("ssrc:", ssrc)
			}
			rtpPort, match := s.getValue(line, "rtpPort:", "$")
			if match {
				log.Println("rtpPort:", rtpPort)
			}
			/*
			/*
				keywords := []KeyWord{
					{
						Key:   "rtpIp",
						Start: "rtpAccessIp",
						End:   "callId",
					},
					{
						Key:   "callId",
						Start: ": ",
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
					log.Println(err)
					return err
				}
				if len(results) > 0 {
					log.Println(line)
					log.Println(results)
				}
		*/
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

func (s *Parser) Run() error {
	s.inviteBye()
	return nil
}

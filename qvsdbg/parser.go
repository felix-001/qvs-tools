package main

import (
	"fmt"
	"log"
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
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/%s/_package/run;grep -E '%s' * -nR\\\"\"", node, service, re)
	log.Println(cmd)
	return RunCmd(cmd)
}

func (s *Parser) inviteBye() error {
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
		log.Println(res)
	}
	return nil
}

func (s *Parser) Run() error {
	s.inviteBye()
	return nil
}

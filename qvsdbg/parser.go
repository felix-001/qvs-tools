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
	cmd := fmt.Sprintf("qssh %s \"cd /home/qboxserver/%s/_package/run;grep -E '%s' * -nR\"", node, service, re)
	log.Println(cmd)
	return RunCmd(cmd)
}

func (s *Parser) Run() error {
	res, err := s.searchLogs("vdn-gdgzh-dls-1-11", "qvs-rtp", "got.*connection")
	if err != nil {
		log.Println(res, err)
		return err
	}
	log.Println(res)
	return nil
}

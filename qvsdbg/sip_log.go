package main

import (
	"fmt"
	"log"
	"strings"
)

func FetchSipMsg(node, instance, params string) (string, error) {
	log.Println("fetching sip log", node)
	if _, err := writeScriptToNode(node); err != nil {
		log.Println("write script err", node)
		return "", err
	}
	service := "qvs-sip"
	if instance != "" {
		service += instance
	}
	path := fmt.Sprintf("/home/qboxserver/%s/_package/run/auditlog/sip_dump", service)
	args := []string{
		path,
		params,
	}
	msg, err := runMultiLineSearchScript(node, args)
	if err != nil {
		log.Println("run script err", node, args)
		return "", err
	}
	//log.Println("msg:", msg, "len:", len(msg))
	if msg == sep {
		return "", fmt.Errorf("not found")
	}
	log.Println("fetch sip log", node, "done")
	return msg, nil
}

func (s *Parser) SearchSipLogs() {
	if s.Conf.Node == "" {
		log.Println("need node, ex: jjh1445_2")
		return
	}
	ss := strings.Split(s.Conf.Node, "_")
	node := ss[0]
	instance := ""
	if len(ss) == 2 {
		instance = ss[1]
	}
	result, err := FetchSipMsg(node, instance, s.Conf.Keywords)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(result)
}

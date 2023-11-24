package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func (s *Parser) httpProc(input string) string {
	ss := strings.Split(input, " ")
	cmd := ss[0]
	keywords := ss[1]
	node := ss[2]
	switch cmd {
	case "sip":
		s.Conf.Keywords = keywords
		s.Conf.Node = node
		return s.SearchSipLogs()
	}
	return ""
}

func (s *Parser) HttpSrvRun() {
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		// 从前端请求中读取数据
		data := r.FormValue("data")
		result := s.httpProc(data)
		fmt.Fprintf(w, result)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 将处理后的数据作为响应发送给前端
		fmt.Fprintf(w, html)
	})

	log.Fatal(http.ListenAndServe(":8000", nil))
}

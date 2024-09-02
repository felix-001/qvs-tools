package main

import "fmt"

func (s *Parser) pathqueryChk() {
	lines := s.loadElkLog(s.conf.PathqueryLogFile)
	for _, line := range lines {
		fmt.Println(line)
	}
}

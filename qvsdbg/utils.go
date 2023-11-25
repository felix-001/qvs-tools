package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

func insertString(original, insert string, pos int) string {
	if pos < 0 || pos > len(original) {
		return original
	}

	return original[:pos] + insert + original[pos:]
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

func (s *Parser) multiLineQuery(Keywords []string) (query string) {
	//query = "(?s)(?<=<--------------------------------------------------------------------------------------------------->).*?"
	query = "(?s)(---).*?"
	for _, keyword := range Keywords {
		query += keyword + ".*?"
	}
	query += "(---)"
	return
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

func (s *Parser) getFirstLogAfterTimePoint(logs, t string) (string, error) {
	ss := strings.Split(logs, "\r\n")
	if len(ss) == 1 {
		return logs, nil
	}
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
		if dateTime > t {
			return str, nil
		}
	}

	return "", fmt.Errorf("log not found")
}

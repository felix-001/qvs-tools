package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func SplitLogLine(line string) []string {
	var result []string
	var buffer bytes.Buffer
	inBrace := false
	braceDepth := 0

	for i := 0; i < len(line); {
		r := rune(line[i])

		// 处理转义字符（如\u0026）
		if r == '\\' && i+5 < len(line) && line[i+1] == 'u' {
			buffer.WriteString(line[i : i+6])
			i += 6
			continue
		}

		switch {
		case r == '{':
			if !inBrace {
				inBrace = true
			}
			braceDepth++
			buffer.WriteRune(r)
			i++
		case r == '}':
			braceDepth--
			buffer.WriteRune(r)
			if braceDepth == 0 {
				inBrace = false
				// 捕获完整JSON块
				result = append(result, buffer.String())
				buffer.Reset()
			}
			i++
		case !inBrace && (r == ' ' || r == '\t'):
			// 处理普通字段
			if buffer.Len() > 0 {
				result = append(result, buffer.String())
				buffer.Reset()
			}
			// 跳过连续空格
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
		default:
			buffer.WriteRune(r)
			i++
		}
	}

	// 处理最后一个字段
	if buffer.Len() > 0 {
		result = append(result, buffer.String())
	}

	return result
}

type LogFields struct {
	T            string
	Path         string
	ReqHeaders   RequestHeaders
	Reqbody      string
	StatusCode   string
	RespHeaders  RespHeaders
	RespBody     string
	RespLength   string
	RespDuration string
}

func fielesToStruct(fields []string) (LogFields, error) {

	result := LogFields{
		T:          fields[2],
		Path:       fields[4],
		ReqHeaders: RequestHeaders{},
	}
	if err := json.Unmarshal([]byte(fields[5]), &result.ReqHeaders); err != nil {
		log.Println(err)
		return result, err
	}
	if len(fields[6]) > 0 && fields[6][0] == '{' {
		result.Reqbody = fields[6]
		result.StatusCode = fields[7]
		result.RespHeaders = RespHeaders{}
		if err := json.Unmarshal([]byte(fields[8]), &result.RespHeaders); err != nil {
			log.Println(err)
			return result, err
		}
		if fields[9][0] == '{' {
			result.RespBody = fields[9]
			result.RespLength = fields[10]
			result.RespDuration = fields[11]
		} else {
			result.RespLength = fields[9]
			result.RespDuration = fields[10]
		}
	} else {
		result.StatusCode = fields[6]
		result.RespHeaders = RespHeaders{}
		if err := json.Unmarshal([]byte(fields[7]), &result.RespHeaders); err != nil {
			log.Println(err)
			return result, err
		}
		if fields[8][0] == '{' {
			result.RespBody = fields[8]
			result.RespLength = fields[9]
			result.RespDuration = fields[10]
		} else {
			result.RespLength = fields[8]
			result.RespDuration = fields[9]
		}
	}
	return result, nil
}

type RequestHeaders struct {
	AcceptEncoding string `json:"Accept-Encoding"`
	ContentLength  string `json:"Content-Length"`
	ContentType    string `json:"Content-Type"`
	Host           string `json:"Host"`
	IP             string `json:"IP"`
	Token          struct {
		Appid int `json:"appid"`
		Uid   int `json:"uid"`
		Utype int `json:"utype"`
	} `json:"Token"`
	UserAgent     string `json:"User-Agent"`
	XForwardedFor string `json:"X-Forwarded-For"`
	XRealIp       string `json:"X-Real-Ip"`
	XReqid        string `json:"X-Reqid"`
	XScheme       string `json:"X-Scheme"`
}

type RespHeaders struct {
	ContentLength string   `json:"Content-Length"`
	ContentType   string   `json:"Content-Type"`
	XLog          []string `json:"X-Log"`
	XReqid        string   `json:"X-Reqid"`
}

func (s *Parser) Qps() {
	if s.conf.F == "" {
		log.Println("input file empty, need -f xxx")
		return
	}
	file, err := os.Open(s.conf.F)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	uidLogs := make([]LogFields, 0)

	var duration int64 = 0
	var start int64 = 0
	totalLine := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		totalLine++
		if line == "" {
			continue
		}
		fields := SplitLogLine(line)
		if len(fields) < 10 {
			log.Println("invalid line:", line, len(fields))
			continue
		}

		logFields, err := fielesToStruct(fields)
		if err != nil {
			log.Println(err, line)
			continue
		}
		if start == 0 {
			start, _ = strconv.ParseInt(logFields.T, 10, 64)
		} else {
			cur, _ := strconv.ParseInt(logFields.T, 10, 64)
			duration = cur - start
		}
		if fmt.Sprintf("%d", logFields.ReqHeaders.Token.Uid) != s.conf.Uid {
			continue
		}
		if !strings.Contains(logFields.Path, "sipraw") {
			continue
		}

		cur, _ := strconv.ParseInt(logFields.T, 10, 64)
		duration = cur - start
		if duration >= 1e7 {
			log.Println("qps: ", len(uidLogs))
			start = cur
			uidLogs = make([]LogFields, 0)
		}

		uidLogs = append(uidLogs, logFields)
	}
	log.Println("log cnt:", len(uidLogs), "total line:", totalLine, "duration:", duration/1e7, "qps:", float64(len(uidLogs))/float64(duration/1e7))

}

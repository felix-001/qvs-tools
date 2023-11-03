package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jart/gosip/sip"
	"golang.org/x/net/html/charset"
)

var (
	gbid  = ""
	total = 0
)

const (
	sep         = "<--------------------------------------------------------------------------------------------------->"
	startPrefix = "message:"
)

type GBPaser struct {
	keepaliveTime int64
}

func (s *GBPaser) onResp(msg *sip.Msg) error {
	return nil
}

type Item struct {
	ChId         string `xml:"DeviceID"`
	Name         string `xml:"Name"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
}

type DeviceList struct {
	Num   string `xml:"Num,attr"`
	Items []Item `xml:"Item"`
}

type XmlMsg struct {
	CmdType    string     `xml:"CmdType"`
	SN         string     `xml:"SN"`
	DeviceId   string     `xml:"DeviceID"`
	SumNum     int        `xml:"SumNum"`
	DeviceList DeviceList `xml:"DeviceList,omitempty"`
}

func parseXml(raw string) (*XmlMsg, error) {
	xmlMsg := &XmlMsg{}
	decoder := xml.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(xmlMsg); err != nil {
		return xmlMsg, err
	}
	return xmlMsg, nil
}

func (s *GBPaser) onKeepalive(msg *sip.Msg, t int64) error {
	if s.keepaliveTime == 0 {
		log.Println("keepalive time:", s.keepaliveTime)
		s.keepaliveTime = t
		return nil
	}
	log.Println("keepalive inerval:", t-s.keepaliveTime)
	s.keepaliveTime = t
	return nil
}

func (s *GBPaser) onMessage(msg *sip.Msg, t int64) error {
	if gbid != msg.From.Uri.User {
		//log.Println("user:", msg.From.Uri.User)
		return nil
	}
	//log.Println("in")
	if !strings.EqualFold(msg.Payload.ContentType(), "application/MANSCDP+xml") {
		log.Println("收到消息格式为非xml,暂不处理", msg.String(), msg.Payload.ContentType())
		return nil
	}
	xmlMsg, err := parseXml(string(msg.Payload.Data()))
	if err != nil {
		return err
	}
	switch xmlMsg.CmdType {
	case "Catalog":
	case "Keepalive":
		return s.onKeepalive(msg, t)
	case "Alarm":
	default:
		log.Println("unknow cmdtype:", xmlMsg.CmdType)
	}
	return nil
}

func (s *GBPaser) onSIP(msg *sip.Msg, t int64) error {
	if msg.IsResponse() {
		return s.onResp(msg)
	}
	switch msg.Method {
	case "REGISTER":
		//log.Println("user:", msg.From.Uri.User)
	case "MESSAGE":
		return s.onMessage(msg, t)
	case "INVITE":
	case "ACK":
	case "BYE":
	case "SUBSCRIBE":
	case "OPTIONS":
	default:
		log.Println("未处理的方法:", msg.Method)
	}
	return nil
}

func str2unix(s string) (int64, error) {
	loc, _ := time.LoadLocation("Local")
	the_time, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return the_time.UnixMilli(), nil
}

func (s *GBPaser) getTime(line string) int64 {
	start := strings.Index(line, "[")
	if start < 0 {
		log.Println("find [ err, raw:", line)
		return -1
	}
	line = line[start+1:]
	end := strings.Index(line, "]")
	if end < 0 {
		log.Println("find ] err, raw:", line)
		return -1
	}
	line = line[:end-4] //4: 毫秒部分
	t, err := str2unix(line)
	if err != nil {
		log.Println("str2unix err:", err, "raw:", line)
		return -1
	}
	return t / 1000
}

func MyScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0 : i+1], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func processOneSipMsg(raw, line string) error {
	raw = strings.ReplaceAll(raw, "tag=tag=", "tag=")
	if len(line) > len(sep)+1 {
		start := strings.Index(line, "<--")
		if start == -1 {
			log.Println("get sep err: ", line)
			return fmt.Errorf("get sep err: %v", line)
		}
		if start > 0 {
			raw += line[:start]
		} else {
			log.Println("err line:", line, len(line), len(sep))
		}
	}
	total++
	if len(raw) == 4 {
		raw = ""
		continue
	}
	msg, err := sip.ParseMsg([]byte(raw))
	if err != nil {
		//log.Printf("lineNo: %d err: %v str: %s hex: %#x\n", lineNo, err, raw, raw)
		errcnt++
		raw = ""
		continue
	}
	if err := s.onSIP(msg, t); err != nil {
		panic(err)
	}
}

/*
* 解析qvs sip_msg_dmmp文件
* 信令转换成结构化数据, json/csv
 */

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	f := os.Args[1]
	gbid = os.Args[2]
	if gbid == "" {
		log.Printf("usage: ./gb-parse <log-file> <gbid>")
		return
	}
	file, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	parser := GBPaser{}
	raw := ""
	scanner := bufio.NewScanner(file)
	errcnt := 0
	lineNo := 0
	var t int64 = 0
	scanner.Split(MyScanLines)
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if strings.Contains(line, "nbbuf") || strings.Contains(line, "send_message") {
			t = parser.getTime(line)
			if t < 0 {
				log.Println("get time err:", line)
				return
			}
			continue
		}
		if strings.Contains(line, "recv message") {
			continue
		}
		if strings.Contains(line, sep) {
			processOneSipMsg(raw, line)
			raw = ""
			continue
		}
		raw += line
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	log.Println("errcnt:", errcnt)
	log.Println("total:", total)
}

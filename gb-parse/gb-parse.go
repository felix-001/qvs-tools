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
	gbid = ""
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
	if msg.Payload.ContentType() != "application/MANSCDP+xml" {
		//log.Println("收到消息格式为非xml,暂不处理", msg.String())
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
	return t
}

func getSipRawMsg(s string) (string, error) {
	start := strings.Index(s, startPrefix)
	if start == -1 {
		return "", fmt.Errorf("get sip raw msg, find start prefix err: %s", s)
	}
	s = s[start+len(startPrefix)+1:]
	end := strings.Index(s, sep)
	if start == -1 {
		return "", fmt.Errorf("get sip raw msg, find sep err: %s", s)
	}
	s = s[:end]
	return s, nil
}

func MyScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		//return i + 1, data[0:i], nil
		return i + 1, data[0 : i+1], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

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
	total := 0
	var t int64 = 0
	scanner.Split(MyScanLines)
	for scanner.Scan() {
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
			/*
				if total > 2 {
					return
				}
			*/
			if len(line) > len(sep) {
				start := strings.Index(line, "<--")
				if start == -1 {
					log.Println("get sep err: ", line)
					break
				}
				if start > 0 {
					raw += line[:start]
				} else {
					log.Println("err line:", line)
				}
			}
			total++
			//log.Println("raw:", raw)
			if len(raw) == 4 {
				raw = ""
				continue
			}
			//if !strings.Contains(line, "Content-Length : 0") {
			//raw += "\r\n"
			//}
			msg, err := sip.ParseMsg([]byte(raw))
			if err != nil {
				log.Println(err, len(raw), raw)
				log.Printf("%#x\n", raw)
				errcnt++
				raw = ""
				//panic(err)
				continue
			}
			if err := parser.onSIP(msg, t); err != nil {
				panic(err)
			}
			raw = ""
			continue
		}
		//if line == "" {
		//	continue
		//}
		//if strings.Contains(line, "Content-Length") {
		//log.Println("1")
		//line += "\r\n"
		//}
		//log.Println("line:", line)
		//log.Printf("line hex: %#x\n", line)
		//raw += line + "\r\n"
		raw += line
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	log.Println("errcnt:", errcnt)
	log.Println("total:", total)
}

func main1() {
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

	errcnt := 0
	total := 0
	parser := GBPaser{}
	var b []byte = make([]byte, 1)
	raw := []byte{}
	for {
		if _, err := file.Read(b); err != nil {
			log.Println(err)
			break
		}
		raw = append(raw, b[0])
		s := string(raw)
		if strings.Contains(s, sep) {
			//log.Println("total:", total)
			t := parser.getTime(s)
			if t < 0 {
				log.Println("get time err:", s)
				return
			}
			sipMsg, err := getSipRawMsg(s)
			if err != nil {
				break
			}
			if len(sipMsg) < 10 {
				raw = raw[:0]
				continue
			}
			msg, err := sip.ParseMsg([]byte(sipMsg))
			if err != nil {
				//log.Println(err, len(raw), string(sipMsg))
				//log.Printf("%#x len:%d raw:%s\n", sipMsg, len(sipMsg), sipMsg)
				errcnt++
				log.Println("errcnt:", errcnt, "total:", total)
				raw = raw[:0]
				//panic(err)
				continue
			}
			if err := parser.onSIP(msg, t); err != nil {
				panic(err)
			}
			total++

			raw = raw[:0]
		}
	}
	log.Println("errcnt:", errcnt)
	log.Println("total:", total)
}

func main3() {
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

	scanner := bufio.NewScanner(file)
	cnt := 0
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("line: %s hex: %#x\n", line, line)
		cnt++
		if cnt == 30 {
			break
		}
	}
}

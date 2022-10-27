package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	baseUrl = "http://qvs-pdr.qnlinking.com/api/v1/jobs"
	conf    = "/usr/local/etc/pdr.conf"
)

type Pdr struct {
	token string
	start int64
	end   int64
	gbId  string
	chId  string
	reqId string
}

type QueryData struct {
	StartTime   int64  `json:"startTime"`
	EndTime     int64  `json:"endTime"`
	Query       string `json:"query"`
	CollectSize int    `json:"collectSize"`
}

func httpReq(method, addr, body string, headers map[string]string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		log.Println("status code", resp.StatusCode, string(respBody))
		return nil, errors.New("http status code err")
	}
	return respBody, err
}

func getToken() (string, error) {
	b, err := ioutil.ReadFile(conf)
	if err != nil {
		log.Println("read fail", conf, err)
		return "", err
	}
	return string(b)[:len(string(b))-1], nil
}

func (self *Pdr) createJob(query string) (string, error) {
	queryData := &QueryData{
		StartTime:   self.start,
		EndTime:     self.end,
		Query:       query,
		CollectSize: 500000,
	}
	data, err := json.Marshal(queryData)
	if err != nil {
		log.Println(err)
		return "", err
	}
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": self.token,
	}
	respBody, err := httpReq("POST", baseUrl, string(data), headers)
	if err != nil {
		return "", err
	}
	res := &struct {
		Id string `json:"id"`
	}{}
	if err = json.Unmarshal(respBody, res); err != nil {
		log.Println(err)
		return "", err
	}
	return res.Id, err
}

func (self *Pdr) isJobDone(jobId string) (bool, error) {
	addr := baseUrl + "/" + jobId
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": self.token,
	}
	respBody, err := httpReq("GET", addr, "", headers)
	if err != nil {
		return false, err
	}
	res := &struct {
		Process int `json:"process"`
	}{}
	if err := json.Unmarshal(respBody, res); err != nil {
		return false, err
	}
	return res.Process == 1, nil
}

func (self *Pdr) waitJobDone(jobId string) error {
	for {
		done, err := self.isJobDone(jobId)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (self *Pdr) pdrGet(addr string) ([]byte, error) {
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": self.token,
	}
	return httpReq("GET", addr, "", headers)
}

func (self *Pdr) downloadLog(jobId string) (string, string, string, error) {
	addr := baseUrl + "/" + jobId + "/events?rawLenLimit=false&pageSize=5000000&prefix=&order=desc&sort=updateTime"
	respBody, err := self.pdrGet(addr)
	if err != nil {
		return "", "", "", err
	}
	//log.Println(string(respBody))
	res := &struct {
		Rows []struct {
			Raw struct {
				Value string `json:"value"`
			} `json:"_raw"`
			Host struct {
				Value string `json:"value"`
			} `json:"host"`
			Origin struct {
				Value string `json:"value"`
			} `json:"origin"`
		} `json:"rows"`
	}{}
	if err := json.Unmarshal(respBody, res); err != nil {
		return "", "", "", err
	}
	raw := ""
	for _, row := range res.Rows {
		raw += row.Raw.Value + "\n"
	}
	if len(raw) == 0 {
		return "", "", "", fmt.Errorf("no log found")
	}
	//log.Println(string(respBody))
	return res.Rows[0].Host.Value, res.Rows[0].Origin.Value, raw, nil
}

func (self *Pdr) getLog(query string) (string, string, string, error) {
	jobId, err := self.createJob(query)
	if err != nil {
		return "", "", "", err
	}
	//log.Println("jobId:", jobId)
	if err := self.waitJobDone(jobId); err != nil {
		return "", "", "", err
	}
	//log.Println("wait done")
	return self.downloadLog(jobId)
}

func NewPdr(reqId, gbId, chId, token string, start, end int64) *Pdr {
	return &Pdr{gbId: gbId, chId: chId, reqId: reqId, token: token, start: start, end: end}
}

func (self *Pdr) getLogTime(s string) string {
	s = strings.ReplaceAll(s, "0m", "")
	s = strings.ReplaceAll(s, "31m", "")
	re := regexp.MustCompile(`\[(.*?)\]`)
	submatchall := re.FindAllString(s, -1)
	element := submatchall[0]
	element = strings.Trim(element, "[")
	element = strings.Trim(element, "]")
	return element
}

func (self *Pdr) getSSRC() (string, error) {
	query := fmt.Sprintf("repo=\"logs\" \"sip_invite\" \"%s\"", self.reqId)
	host, _, data, err := self.getLog(query)
	if err != nil {
		return "", err
	}
	log.Println("qvs-sip node:", host)
	//log.Println("log:", data)
	log.Println("log time:", self.getLogTime(data))
	rtpSrvIP, err := self.getVal(data, "ip=", "&reqId")
	if err != nil {
		return "", err
	}
	rtpPort, err := self.getVal(data, "rtp_port=", "&")
	if err != nil {
		return "", err
	}
	log.Println("rtp service ip:", rtpSrvIP)
	log.Println("rtp service port:", rtpPort)
	return self.getVal(data, "ssrc=", "&")
}

func (self *Pdr) getCallID(ssrc string) (string, error) {
	query := fmt.Sprintf("repo=\"logs\" \"return callid\" \"%s\"", ssrc)
	_, _, data, err := self.getLog(query)
	if err != nil {
		return "", err
	}
	//log.Println("log:", data)
	return self.getVal(data, "callid:", "\n")
}

func (self *Pdr) getVal(origin, startPrefix, endPrefix string) (string, error) {
	start := strings.Index(origin, startPrefix)
	if start == -1 {
		return "", fmt.Errorf("can't find %s from %s", startPrefix, origin)
	}
	start += len(startPrefix)
	end := len(origin)
	if endPrefix != "" {
		end = strings.Index(origin[start:], endPrefix)
		if end == -1 {
			return "", fmt.Errorf("can't find %s from %s err", endPrefix, origin)
		}
	}
	return origin[start : start+end], nil
}

func (self *Pdr) isInviteRespOK(callid string) (bool, error) {
	query := fmt.Sprintf("repo=\"logs\" \"200\" \"callid=%s\"", callid)
	//log.Println(query)
	_, _, data, err := self.getLog(query)
	if err != nil {
		return false, err
	}
	//log.Println("log:", data)
	if len(data) == 0 {
		return false, nil
	}
	return true, nil
}

func (self *Pdr) IsDevTcpConnect() (bool, error) {
	query := fmt.Sprintf("repo=\"logs\" \"tcp attach\" \"%s\"", self.reqId)
	log.Println(query)
	host, origin, data, err := self.getLog(query)
	if err != nil {
		return false, err
	}
	log.Println("host", host, "origin", origin)
	log.Println(data)
	if len(data) == 0 {
		return false, nil
	}
	return true, nil
}

func (self *Pdr) liveStreamDbg() error {
	ssrc, err := self.getSSRC()
	if err != nil {
		return err
	}
	log.Println("ssrc:", ssrc)
	callid, err := self.getCallID(ssrc)
	if err != nil {
		return err
	}
	log.Println("callid:", callid)
	ok, err := self.isInviteRespOK(callid)
	if err != nil {
		return err
	}
	log.Println("invite resp 200 ok:", ok)
	conn, err := self.IsDevTcpConnect()
	if err != nil {
		return err
	}
	log.Println("device tcp connect:", conn)
	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)

	dbgType := flag.String("type", "live", "type")
	reqId := flag.String("reqid", "", "reqId")
	gbId := flag.String("gbid", "", "gbId")
	chId := flag.String("chid", "", "chId")
	end := flag.Int64("end", time.Now().UnixMilli(), "end")
	start := flag.Int64("start", *end-3600*24*1000, "start")
	flag.Parse()
	if *reqId == "" {
		log.Println("err: need reqId")
		flag.PrintDefaults()
		return
	}
	if *gbId == "" {
		log.Println("err: need gbId")
		flag.PrintDefaults()
		return
	}

	token, err := getToken()
	if err != nil {
		log.Fatalln("get token err", err)
	}
	pdr := NewPdr(*reqId, *gbId, *chId, token, *start, *end)
	switch *dbgType {
	case "live":
		if err := pdr.liveStreamDbg(); err != nil {
			log.Fatalln("live stream dbg err:", err)
		}
	}
}

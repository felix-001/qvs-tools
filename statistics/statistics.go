package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ak    string
	sk    string
	host  string
	defAK string
	defSK string
	csv   string
)

var (
	errHttpStatusCode = errors.New("http status code err")
)

const (
	conf = "/usr/local/etc/statistics.conf"
)

func hmacSha1(key, data string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(data))
	hm := mac.Sum(nil)
	s := base64.URLEncoding.EncodeToString(hm)
	return s
}

func signToken(ak, sk, method, path, host, body string, headers map[string]string) string {
	data := method + " " + path + "\n"
	data += "Host: " + host
	if headers != nil {
		for key, value := range headers {
			data += "\n" + key + ": " + value
		}
	}
	data += "\n\n"
	if body != "" {
		data += body
	}
	//log.Println("data:")
	//fmt.Println(data)
	token := "Qiniu " + ak + ":" + hmacSha1(sk, data)
	//log.Println("token:", token)
	return token
}

func httpReq(method, addr, body string, headers map[string]string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	//log.Print("resp body", string(resp_body))
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode != 200 {
		log.Println("status code", resp.StatusCode, string(resp_body))
		return "", errHttpStatusCode
	}
	return string(resp_body), err
}

func qvsHttpReq(method, addr, body string) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		log.Println(err)
		return "", err
	}
	host := u.Host
	u.Host = ""
	u.Scheme = ""
	headers := map[string]string{}
	if body != "" {
		headers["Content-Type"] = "application/json"
	}
	//token := signToken(ak, sk, method, u.Path, u.Host, body, headers)
	token := signToken(ak, sk, method, u.String(), host, body, headers)
	headers["Authorization"] = token
	return httpReq(method, addr, body, headers)
}

func qvsGet(path string) (string, error) {
	addr := fmt.Sprintf("%s%s", host, path)
	return qvsHttpReq("GET", addr, "")
}

func parseConf() {
	b, err := ioutil.ReadFile(conf)
	if err != nil {
		log.Printf("%s not found\n", conf)
		return
	}
	keys := struct {
		AK   string `json:"ak"`
		SK   string `json:"sk"`
		HOST string `json:"host"`
	}{}
	if err := json.Unmarshal(b, &keys); err != nil {
		log.Println("parse conf err", err)
		return
	}
	defAK = keys.AK
	defSK = keys.SK
	host = keys.HOST
}

func parseConsole() {
	flag.StringVar(&ak, "ak", defAK, "ak")
	flag.StringVar(&sk, "sk", defSK, "sk")
	flag.StringVar(&host, "host", host, "host")
	flag.StringVar(&csv, "csv", "", "csv file")
	flag.Parse()
	if ak == "" {
		log.Println("need ak")
		flag.PrintDefaults()
		os.Exit(0)
	}
	if sk == "" {
		log.Println("need sk")
		flag.PrintDefaults()
		os.Exit(0)
	}
	if host == "" {
		log.Println("need host")
		flag.PrintDefaults()
		os.Exit(0)
	}
}

type StatisticsInstance struct {
}

func (s *StatisticsInstance) getDevCount() (error, int) {
	return nil, 0
}

type Namespace struct {
	Uid        int    `json:"uid"`
	AccessType string `json:"accessType"`
	ID         string `json:"id"`
}

type NsList struct {
	Items []Namespace `json:"items"`
	Total int         `json:"total"`
}

var allNamespaces []Namespace

func (s *StatisticsInstance) getNsList(offset int) (*NsList, error) {
	path := fmt.Sprintf("/v1/namespaces?line=500&offset=%d", offset)
	resp, err := qvsGet(path)
	if err != nil {
		return nil, err
	}
	//log.Println(resp)
	nsList := NsList{}
	if err := json.Unmarshal([]byte(resp), &nsList); err != nil {
		return nil, err
	}
	//log.Println("total:", nsList.Total, "count:", len(nsList.Items))
	return &nsList, nil
}

func (s *StatisticsInstance) isUidExist(uids []int, uid int) bool {
	for _, v := range uids {
		if v == uid {
			return true
		}
	}
	return false
}

func (s *StatisticsInstance) mergeUids(uids *[]int, nslist *NsList) {
	for _, v := range nslist.Items {
		if v.AccessType == "gb28181" && !s.isUidExist(*uids, v.Uid) {
			*uids = append(*uids, v.Uid)
		}
	}
}

func (s *StatisticsInstance) getUids() ([]int, error) {
	uids := []int{}
	nslist, err := s.getNsList(0)
	if err != nil {
		return nil, err
	}
	s.mergeUids(&uids, nslist)
	log.Println("total namespaces:", nslist.Total)
	for i := len(nslist.Items); i < nslist.Total; {
		log.Println("total:", nslist.Total, "offset:", i)
		nslist, err := s.getNsList(i)
		if err != nil {
			return nil, err
		}
		s.mergeUids(&uids, nslist)
		i += len(nslist.Items)
	}
	log.Println("total uids:", len(uids))
	return uids, nil
}

type Device struct {
	Gbid     string `json:"gbId"`
	Channels int    `json:"channels"`
	State    string `json:"state"`
}

type DeviceList struct {
	Items []Device `json:"items"`
	Total int      `json:"total"`
}

func (s *StatisticsInstance) getDevicesByUid(uid int) (*DeviceList, error) {
	path := fmt.Sprintf("/v1/devices?uid=%d", uid)
	resp, err := qvsGet(path)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	devlist := DeviceList{}
	if err := json.Unmarshal([]byte(resp), &devlist); err != nil {
		log.Println(err)
		return nil, err
	}
	return &devlist, nil
}

func (s *StatisticsInstance) getDevices(offset int) (*DeviceList, error) {
	path := fmt.Sprintf("/v1/devices?offset=%d&line=1000", offset)
	resp, err := qvsGet(path)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	devlist := DeviceList{}
	if err := json.Unmarshal([]byte(resp), &devlist); err != nil {
		log.Println(err)
		return nil, err
	}
	return &devlist, nil
}

func (s *StatisticsInstance) filterUid(uids []int) (int, error) {
	noDevCnt := 0
	for _, uid := range uids {
		devlist, err := s.getDevicesByUid(uid)
		if err != nil {
			log.Println(err)
			return 0, err
		}
		if devlist.Total == 0 {
			//log.Println("uid:", uid, "no devices")
			noDevCnt++
		}

	}
	return noDevCnt, nil
}

func (s *StatisticsInstance) get(path string, v interface{}) error {
	resp, err := qvsGet(path)
	if err != nil {
		log.Println(err)
		return err
	}
	if err := json.Unmarshal([]byte(resp), v); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

type Stream struct {
	Status       bool   `json:"status"`
	StreamId     string `json:"streamId"`
	LastPushedAt int64  `json:"lastPushedAt"`
	NamespaceId  string `json:"nsId"`
	UID          int    `json:"uid"`
}

type StreamList struct {
	Items []Stream `json:"items"`
	Total int      `json:"total"`
}

// /v1/streams?prefix=31011500991320000384&namespaceId=bj&qtype=0&line=10&offset=0
func (s *StatisticsInstance) getStreams(nsId string, line, offset int) (*StreamList, error) {
	path := fmt.Sprintf("/v1/streams?namespaceId=%s&line=%d&offset=%d&qtype=0", nsId, line, offset)
	log.Println(path)
	resp, err := qvsGet(path)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	streamlist := StreamList{}
	if err := json.Unmarshal([]byte(resp), &streamlist); err != nil {
		log.Println(err)
		return nil, err
	}
	//log.Println("total streams:", streamlist.Total, "nsId:", nsId)
	return &streamlist, nil
}

func (s *StatisticsInstance) getAllStreamsByNamespace(nsId string) (streams []Stream, err error) {
	streamlist, err := s.getStreams(nsId, 1000, 0)
	if err != nil {
		return nil, err
	}
	log.Println("nsid:", nsId, "total:", streamlist.Total, "len:", len(streamlist.Items))
	streams = append(streams, streamlist.Items...)
	if streamlist.Total < 1000 {
		return
	}
	for i := len(streamlist.Items); i < streamlist.Total; i += len(streamlist.Items) {
		streamlist, err = s.getStreams(nsId, 1000, i)
		if err != nil {
			return nil, err
		}
		streams = append(streams, streamlist.Items...)
		if len(streamlist.Items) < 1000 {
			break
		}

	}
	return
}

func (s *StatisticsInstance) getAllRtmpStreams(namespaces []Namespace) (streams []Stream, err error) {
	for i, ns := range namespaces {
		log.Println("nsid:", ns.ID)
		stremlist, err := s.getAllStreamsByNamespace(ns.ID)
		if err != nil {
			return nil, err
		}
		log.Println("i:", i, "ns:", ns.ID, "streams:", len(stremlist))
		streams = append(streams, stremlist...)
		//time.Sleep(time.Second)
	}
	return
}

func (s *StatisticsInstance) getDevCountByUID(uid int) {
	devlist := &DeviceList{}
	path := fmt.Sprintf("/v1/devices?uid=%d&line=1000", uid)
	if err := s.get(path, devlist); err != nil {
		panic(err)
	}
	total := 0
	for _, dev := range devlist.Items {
		total += dev.Channels
	}
	log.Println("total:", total, "raw total:", devlist.Total, "item count:", len(devlist.Items))
}

func (s *StatisticsInstance) getTotalDevCnt() error {
	devlist, err := s.getDevices(0)
	if err != nil {
		return err
	}
	total := 0
	for _, dev := range devlist.Items {
		total += dev.Channels
	}
	for i := len(devlist.Items); i < devlist.Total; {
		devlist, err := s.getDevices(i)
		if err != nil {
			return err
		}
		for _, dev := range devlist.Items {
			if dev.State != "online" {
				continue
			}
			total += dev.Channels
		}
		i += len(devlist.Items)

	}
	log.Println("total:", total)
	return nil
}

func (s *StatisticsInstance) getAllNamespaces() {
	namespaces, err := s.getNsList(0)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("total:", namespaces.Total)
	allNamespaces = append(allNamespaces, namespaces.Items...)
	for i := len(namespaces.Items); i < namespaces.Total; {
		log.Println("i:", i)
		namespaces, err := s.getNsList(i)
		if err != nil {
			log.Fatalln(err)
		}
		allNamespaces = append(allNamespaces, namespaces.Items...)
		i += len(namespaces.Items)
	}
}

var allRtmpNamespaces []Namespace

func (s *StatisticsInstance) getAllRtmpNamespaces() {
	s.getAllNamespaces()
	for _, ns := range allNamespaces {
		if ns.AccessType == "rtmp" {
			allRtmpNamespaces = append(allRtmpNamespaces, ns)
		}
	}
	log.Println("rtmp namespaces:", len(allRtmpNamespaces))
}

type Template struct {
	ID           string `json:"id"`
	RecordType   int    `json:"recordType"`   // 录制模式，取值：0（不录制），1（实时录制），2（按需录制）
	TemplateType int    `json:"templateType"` // 模板类型，取值：0（录制模版），1（截图模版）
}

// /v1/templates?templateType=0&uid=1380505636&line=10&offset=0
func (s *StatisticsInstance) getTemplate(uid string) ([]Template, error) {
	temps := struct {
		Items []Template `json:"items"`
		Total int        `json:"total"`
	}{}
	path := fmt.Sprintf("/v1/templates?uid=%s&line=1000&offset=0&templateType=0", uid)
	if err := s.get(path, &temps); err != nil {
		panic(err)
	}
	return temps.Items, nil
}

func (s *StatisticsInstance) getAllStorageFee() {
	b, err := ioutil.ReadFile(csv)
	if err != nil {
		log.Fatalln("read fail", csv, err)
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	totalFee := 0
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if i == 0 {
			i++
			continue
		}
		i++
		ss := strings.Split(line, ",")
		uid := ss[1]
		kodoFeeStr := ss[len(ss)-1]
		kodoFee, err := strconv.ParseInt(kodoFeeStr, 10, 32) // 10 表示十进制，32 表示 int 的位宽（32位）
		if err != nil {
			fmt.Println("Error converting string to int:", err)
			return
		}
		qvsFeeStr := ss[len(ss)-2]
		qvsFee, err := strconv.ParseInt(qvsFeeStr, 10, 32) // 10 表示十进制，32 表示 int 的位宽（32位）
		if err != nil {
			fmt.Println("Error converting string to int:", err)
			return
		}

		if qvsFee > 30 {
			totalFee += int(qvsFee)
		}
		log.Println(uid, kodoFee, qvsFee)
		/*
			tmps, _ := s.getTemplate(uid)
			if len(tmps) > 0 {
				totalFee += int(kodoFee)
			}
		*/
	}
	log.Println("total:", totalFee)
}

func (s *StatisticsInstance) getAllActiveStreams(streams []Stream) (outStreams []Stream) {
	for _, stream := range streams {
		if stream.LastPushedAt != 0 && time.Now().Unix()-stream.LastPushedAt < 3*24*3600 {
			outStreams = append(outStreams, stream)
		}
	}
	return
}

func main() {
	log.SetFlags(log.Lshortfile)
	parseConf()
	parseConsole()
	s := StatisticsInstance{}
	s.getAllRtmpNamespaces()
	allStreams, err := s.getAllRtmpStreams(allRtmpNamespaces)
	log.Println("total stream count:", len(allStreams), "err:", err)
	activeStreams := s.getAllActiveStreams(allStreams)
	log.Println("total active streams:", len(activeStreams))
	jsonbody, err := json.Marshal(activeStreams)
	if err != nil {
		log.Println(err)
		return
	}
	err = ioutil.WriteFile("out.json", jsonbody, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	//s.getAllStorageFee()
	//s.getStreams(allRtmpNamespaces[0].ID, 2000, 0)
	//s.getAllNamespaces()
	//log.Println(len(allNamespaces))
	/*
		uids, err := s.getUids()
		if err != nil {
			panic(err)
		}
		noDevCnt, err := s.filterUid(uids)
		if err != nil {
			panic(err)
		}
		log.Println("no dev uid cnt:", noDevCnt)
	*/
	/*
			streamlist := &StreamList{}
			if err := s.get("/v1/streams?qtype=1", streamlist); err != nil {
				panic(err)
			}
			log.Println("同时在线推流路数:", streamlist.Total)
			streamlist = &StreamList{}
			if err := s.get("/v1/streams", streamlist); err != nil {
				panic(err)
			}
		log.Println("总共流个数:", streamlist.Total)
		s.getDevCountByUID()
	*/
	//s.getTotalDevCnt()
}

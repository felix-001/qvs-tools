package main

import (
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
)

var (
	ak    string
	sk    string
	host  string
	defAK string
	defSK string
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
}

type NsList struct {
	Items []Namespace `json:"items"`
	Total int         `json:"total"`
}

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
	LastPushedAt int    `json:"lastPushedAt"`
}

type StreamList struct {
	Items []Stream `json:"items"`
	Total int      `json:"total"`
}

func (s *StatisticsInstance) getStreams() error {
	path := fmt.Sprintf("/v1/streams")
	resp, err := qvsGet(path)
	if err != nil {
		log.Println(err)
		return err
	}
	devlist := DeviceList{}
	if err := json.Unmarshal([]byte(resp), &devlist); err != nil {
		log.Println(err)
		return err
	}
	return nil
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

func main() {
	log.SetFlags(log.Lshortfile)
	parseConf()
	parseConsole()
	s := StatisticsInstance{}
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
		s.getDevCountByUID(1380463884)
	*/
	s.getTotalDevCnt()
}

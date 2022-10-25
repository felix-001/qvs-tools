package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	addr            string
	start, end      int
	ak, sk          string
	path, body      string
	gbids, nsid     string
	gbid, audioFile string
	get, post       bool
	isDownload      bool
	chid            string
	isStop          bool
	isPlayer        bool
	isTalk          bool
	host            string
)

var (
	errHttpStatusCode = errors.New("http status code err")
)

func hmacSha1(key, data string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(data))
	hm := mac.Sum(nil)
	log.Println(hex.EncodeToString(hm))
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
	log.Println("data:")
	fmt.Println(data)
	token := "Qiniu " + ak + ":" + hmacSha1(sk, data)
	log.Println("token:", token)
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
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode != 200 {
		log.Println("status code", resp.StatusCode, string(resp_body), addr)
		return "", errHttpStatusCode
	}
	return string(resp_body), err
}

func qvsHttpReq(method, addr, body string, headers map[string]string) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		log.Println(err)
		return "", err
	}
	token := signToken(ak, sk, method, u.Path, u.Host, body, headers)
	if headers != nil {
		headers["Authorization"] = token
	}
	return httpReq(method, addr, body, headers)
}

func qvsHttpGet(addr string) (string, error) {
	return qvsHttpReq("GET", addr, "", nil)
}

func httpPost(addr, body string) (string, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	return httpReq("POST", addr, body, headers)
}

func isLocalTest() bool {
	if strings.Contains(host, "100") || strings.Contains(host, "192") || strings.Contains(host, "127") {
		return true
	}
	return false
}

func qvsHttpPost(addr, body string) (string, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	if isLocalTest() {
		headers["authorization"] = "QiniuStub uid=1"
	}
	resp, err := qvsHttpReq("POST", addr, body, headers)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return resp, nil
}

func parseConsole() {
	flag.StringVar(&nsid, "nsid", "", "namespace id")
	flag.StringVar(&host, "host", "qvs.qiniuapi.com", "host")
	flag.StringVar(&gbid, "gbid", "", "gbid")
	flag.StringVar(&chid, "chid", "", "chid")
	flag.StringVar(&gbids, "gbids", "", "gbids")
	flag.StringVar(&addr, "url", "qvs.qiniuapi.com", "url")
	flag.StringVar(&body, "body", "", "body")
	flag.BoolVar(&get, "get", true, "is http get")
	flag.BoolVar(&isTalk, "talk", false, "is talk")
	flag.BoolVar(&isPlayer, "player", false, "gen player url")
	flag.BoolVar(&post, "post", false, "is http post")
	flag.BoolVar(&isDownload, "download", false, "is download")
	flag.BoolVar(&isStop, "stop", false, "is stop")
	flag.StringVar(&audioFile, "audio", "", "audio file")
	flag.StringVar(&path, "path", "", "path")
	flag.StringVar(&ak, "ak", "", "ak")
	flag.StringVar(&sk, "sk", "", "sk")
	flag.IntVar(&start, "start", 0, "start time")
	flag.IntVar(&end, "end", 0, "end time")
	flag.Parse()
}

func qvsGet() (string, error) {
	addr := fmt.Sprintf("http://%s%s", addr, path)
	return qvsHttpGet(addr)
}

func qvsPost(body string) (string, error) {
	u := fmt.Sprintf("http://%s%s", addr, path)
	return qvsHttpPost(u, body)
}

const BlkLen = 320000

func calcBlkLen(len, pos int) int {
	blkLen := BlkLen
	left := len - pos
	if left < BlkLen {
		blkLen = left
	}
	return blkLen
}

func sendBlk(blk []byte, addr string) error {
	base64Blk := base64.StdEncoding.EncodeToString(blk)
	body := fmt.Sprintf("{\"base64_pcm\":\"%s\"}", base64Blk)
	resp, err := httpPost(addr, body)
	if err != nil {
		return err
	}
	log.Println(resp)
	return nil
}

type TalkBody struct {
	TcpModel string   `json:"tcpModel`
	Gbids    []string `json:"gbids"`
}

type TalksResp struct {
	AudioSendAddrForHttp  string `json:"audioSendAddrForHttp"`
	AudioSendAddrForHttps string `json:"audioSendAddrForHttps"`
	Gbid                  string `json:"gbid"`
}

func sendPcm(pcm []byte, addr string) error {
	pos := 0
	for pos < len(pcm) {
		blkLen := calcBlkLen(len(pcm), pos)
		blk := pcm[pos : pos+blkLen]
		err := sendBlk(blk, addr)
		if err != nil {
			return err
		}
		pos += BlkLen
	}
	return nil
}

func sendAudioData(resp string) {
	if audioFile == "" {
		log.Println("err: audioFile need")
		flag.PrintDefaults()
		return
	}
	talkresp := TalksResp{}
	err := json.Unmarshal([]byte(resp), &talkresp)
	if err != nil {
		log.Println(err)
		return
	}
	pcm, err := ioutil.ReadFile(audioFile)
	if err != nil {
		log.Println(err)
		return
	}
	addr := talkresp.AudioSendAddrForHttp
	if isLocalTest() {
		u, err := url.Parse(addr)
		if err != nil {
			log.Fatal("parse url err", addr)
			return
		}
		u.Host += ":2985"
		addr = u.String()
	}
	log.Println("addr:", addr)
	sendPcm(pcm, addr)
}

type Config struct {
	ak  string
	sk  string
	url string
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func loadConf() error {
	file := "/etc/qvs.conf"
	if exists(file) {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			log.Println("read fail", file, err)
			return err
		}
		conf := Config{}
		err = json.Unmarshal(b, &conf)
		if err != nil {
			log.Println(err)
			return err
		}
		ak = conf.ak
		sk = conf.sk
		addr = conf.url
	}
	return nil
}

type QueryUrl struct {
	HttpQueryUrl  string `json:"httpQueryUrl"`
	HttpsQueryUrl string `json:"httpsQueryUrl"`
}

func getQueryUrl() (*QueryUrl, error) {
	if nsid == "" {
		log.Fatal("please input nsid")
	}
	if gbid == "" {
		log.Fatal("please input gbid")
	}
	if start == 0 {
		log.Fatal("please input start")
	}
	if end == 0 {
		log.Fatal("please input end")
	}
	path = fmt.Sprintf("/v1/namespaces/%s/devices/%s/download", nsid, gbid)
	body := &struct {
		ChannelId string `json:"channelId"`
		Start     int    `json:"start"`
		End       int    `json:"end"`
	}{
		Start:     start,
		End:       end,
		ChannelId: chid,
	}
	jsonbody, err := json.Marshal(body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	resp, err := qvsPost(string(jsonbody))
	if err != nil {
		return nil, err
	}
	log.Println(resp)
	queryUrl := &QueryUrl{}
	err = json.Unmarshal([]byte(resp), queryUrl)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return queryUrl, nil
}

type DownloadStatus struct {
	Code int `json:"code"`
	Data struct {
		ID                string `json:"id"`
		Percent           int    `json:"percent"`
		HttpDownloadAddr  string `json:"httpDownloadAddr"`
		HttpsDownloadAddr string `json:"httpsDownloadAddr"`
	}
}

func queryDownloadStatus(queryUrl *QueryUrl) (*DownloadStatus, error) {
	log.Println(queryUrl.HttpQueryUrl)
	resp, err := qvsHttpGet(queryUrl.HttpQueryUrl)
	if err != nil {
		return nil, err
	}
	downloadStatus := &DownloadStatus{}
	err = json.Unmarshal([]byte(resp), downloadStatus)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println(downloadStatus.Code)
	if downloadStatus.Code != 0 {
		return nil, errors.New("code err")
	}
	return downloadStatus, nil
}

func download() error {
	queryUrl, err := getQueryUrl()
	if err != nil {
		log.Println(err)
		return err
	}
	downloadStatus := &DownloadStatus{}
	for downloadStatus.Data.Percent != 100 {
		downloadStatus, err = queryDownloadStatus(queryUrl)
		if err != nil {
			log.Println(err)
			return err
		}
		log.Println("percent:", downloadStatus.Data.Percent)
		time.Sleep(3 * time.Second)
	}
	downloadStatus, err = queryDownloadStatus(queryUrl)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("download url: ", downloadStatus.Data.HttpDownloadAddr)
	return nil
}

func streamStop() error {
	body := &struct {
		Start     int    `json:"start"`
		End       int    `json:"end"`
		ChannelId string `json:"channelId"`
	}{
		Start:     start,
		End:       end,
		ChannelId: chid,
	}
	jsonbody, err := json.Marshal(body)
	if err != nil {
		log.Println(err)
		return err
	}
	path = fmt.Sprintf("/v1/namespaces/%s/devices/%s/stop", nsid, gbid)
	resp, err := qvsPost(string(jsonbody))
	if err != nil {
		return err
	}
	log.Println(resp)
	return nil
}

func hlsPlayer() {
	if addr == "" {
		fmt.Println("please input addr")
		return
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(addr))
	url := "http://236809372.cloudvdn.com:1370/player?video=" + b64
	fmt.Println("url: ", url)
}

func startTalk() {
	if ak == "" {
		log.Fatalln("ak is needed")
	}
	if sk == "" {
		log.Fatalln("sk is needed")
	}
	if host == "" {
		log.Fatalln("host is needed")
	}
	body := &struct {
		IsV2 bool `json:"isV2"`
	}{
		IsV2: true,
	}
	jsonbody, err := json.Marshal(body)
	if err != nil {
		log.Println(err)
		return
	}
	u := fmt.Sprintf("http://%s/v1/namespaces/%s/devices/%s/talk", host, nsid, gbid)
	resp, err := qvsHttpPost(u, string(jsonbody))
	if err != nil {
		log.Fatalln("start talk err", err)
	}
	sendAudioData(resp)

}

func main() {
	log.SetFlags(log.Lshortfile)
	// 首先尝试从文件加载配置
	//loadConf()
	// 控制台指定的参数会覆盖配置文件
	parseConsole()
	if isDownload {
		if err := download(); err != nil {
			log.Println(err)
		}
		return
	}
	if isStop {
		if err := streamStop(); err != nil {
			log.Println(err)
		}
		return
	}
	if isPlayer {
		hlsPlayer()
		return
	}
	if isTalk {
		startTalk()
		return
	}
	if post {
		resp, err := qvsPost(body)
		log.Println(resp, err)
	} else {
		qvsGet()
	}
}

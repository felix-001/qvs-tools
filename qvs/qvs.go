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
	"strings"
)

var ak, sk *string
var nsid, gbid *string
var audioFile *string
var addr *string
var gbids *string
var path *string
var body *string
var get *bool
var post *bool

var (
	errHttpStatusCode = errors.New("http status code err")
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
		log.Println("status code", resp.StatusCode, string(resp_body))
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
	token := signToken(*ak, *sk, method, u.Path, u.Host, body, headers)
	headers["Authorization"] = token
	return httpReq(method, addr, body, headers)
}

func qvsHttpGet(addr string) (string, error) {
	return qvsHttpReq("GET", addr, "", nil)
}

func httpPost(addr, body string) (string, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	return httpReq("POST", addr, body, headers)
}

func qvsHttpPost(addr, body string) (string, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	resp, err := qvsHttpReq("POST", addr, body, headers)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return resp, nil
}

func pm3u8() {
	if *addr == "" {
		flag.PrintDefaults()
		return
	}
	resp, err := qvsHttpGet(*addr)
	if err != nil {
		return
	}
	log.Println(resp)
}

func parseConsole() {
	nsid = flag.String("nsid", "", "namespace id")
	gbid = flag.String("gbid", "", "gbid")
	gbids = flag.String("gbids", "", "gbids")
	addr = flag.String("url", "", "url")
	body = flag.String("body", "", "body")
	get = flag.Bool("get", true, "http get?")
	post = flag.Bool("post", false, "http post?")
	audioFile = flag.String("audiofile", "", "audio file")
	path = flag.String("path", "", "path")
	_ak := flag.String("ak", "", "ak")
	_sk := flag.String("sk", "", "sk")
	_addr := flag.String("addr", "", "addr")
	flag.Parse()
	if *_ak != "" {
		ak = _ak
	}
	if *_sk != "" {
		sk = _sk
	}
	if *_addr != "" {
		addr = _addr
	}
	if path == nil || sk == nil || ak == nil {
		fmt.Println("err: path/ak/sk need")
		flag.PrintDefaults()
		os.Exit(0)
	}
}

func qvsTestGet() {
	addr := fmt.Sprintf("http://qvs-test.qiniuapi.com/v1/%s", *path)
	resp, err := qvsHttpGet(addr)
	if err != nil {
		return
	}
	log.Println(resp)
}

func qvsTestPost(body string) (string, error) {
	addr := fmt.Sprintf("http://qvs-test.qiniuapi.com/v1/%s", *path)
	//addr := fmt.Sprintf("http://qvs.qiniuapi.com/v1/%s", path)
	return qvsHttpPost(addr, body)
}

const BlkLen = 20 * 1024

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

func broadcast() {
	if *gbids == "" || *audioFile == "" || *nsid == "" || *ak == "" || *sk == "" {
		flag.PrintDefaults()
		return
	}
	res := strings.Split(*gbids, " ")
	talkbody := TalkBody{TcpModel: "sendrecv", Gbids: []string{}}
	for _, gbid := range res {
		talkbody.Gbids = append(talkbody.Gbids, gbid)
	}
	jsonbody, err := json.Marshal(talkbody)
	if err != nil {
		log.Println(err)
		return
	}

	resp, err := qvsTestPost(string(jsonbody))
	if err != nil {
		return
	}
	talkresps := []TalksResp{}
	err = json.Unmarshal([]byte(resp), &talkresps)
	if err != nil {
		log.Println(err)
		return
	}
	pcm, err := ioutil.ReadFile(*audioFile)
	if err != nil {
		log.Println(err)
		return
	}
	for _, talkresp := range talkresps {
		go sendPcm(pcm, talkresp.AudioSendAddrForHttp)
	}
}

func createTemplate() {
	jsonbody :=
		`{
			"name":"helloworld111",
			"bucket":"linking",
			"deleteAfterDays":30,
			"fileType":0,
			"recordFileFormat":1,
			"templateType":0,
			"m3u8FileNameTemplate":"${startMs}-${endMs}-${duration}.m3u8"
		}
`
	resp, err := qvsTestPost(jsonbody)
	if err != nil {
		return
	}
	log.Println(resp)
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
		ak = &conf.ak
		sk = &conf.sk
		addr = &conf.url
	}
	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)
	// 首先尝试从文件加载配置
	loadConf()
	// 控制台指定的参数会覆盖配置文件
	parseConsole()
	//broadcast()
	//time.Sleep(60 * time.Second)
	//createTemplate()
	if *post {
		resp, err := qvsTestPost(*body)
		log.Println(resp, err)
	} else {
		qvsTestGet()
	}
}

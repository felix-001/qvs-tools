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
	"strings"
)

var ak, sk *string
var nsid, gbid *string
var audioFile *string
var addr *string
var gbids *string

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
	ak = flag.String("ak", "", "ak")
	sk = flag.String("sk", "", "sk")
	nsid = flag.String("nsid", "", "namespace id")
	gbid = flag.String("gbid", "", "gbid")
	gbids = flag.String("gbids", "", "gbids")
	addr = flag.String("url", "", "pm3u8 url")
	audioFile = flag.String("audiofile", "", "audio file")
	flag.Parse()
}

func qvsTestGet(path string) {
	addr := fmt.Sprintf("http://qvs-test.qiniuapi.com/v1/%s", path)
	resp, err := qvsHttpGet(addr)
	if err != nil {
		return
	}
	log.Println(resp)
}

func qvsTestPost(path, body string) (string, error) {
	addr := fmt.Sprintf("http://qvs-test.qiniuapi.com/v1/%s", path)
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

	resp, err := qvsTestPost(fmt.Sprintf("namespaces/%s/talks", *nsid), string(jsonbody))
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

func main() {
	log.SetFlags(log.Lshortfile)
	parseConsole()
	broadcast()
}

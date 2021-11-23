package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

var ak, sk *string
var nsid, gbid *string
var audioFile *string
var addr *string

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
	log.Println("data:", data)
	token := "Qiniu " + ak + ":" + hmacSha1(sk, data)
	log.Println("token:", token)
	return token
}

func httpPost(ak, sk, host, path, body string, headers map[string]string) ([]byte, error) {
	method := "POST"
	token := signToken(ak, sk, method, path, host, string(body), headers)
	client := &http.Client{}
	req, _ := http.NewRequest(method, "http://"+host+path, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.Header.Add("Authorization", token)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	return resp_body, err
}

func httpReq(method, addr, body string, headers map[string]string) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		log.Println(err)
		return "", err
	}
	token := signToken(*ak, *sk, method, u.Path, u.Host, body, headers)
	client := &http.Client{}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.Header.Add("Authorization", token)
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
	return string(resp_body), err
}

func httpGet(addr string) (string, error) {
	return httpReq("GET", addr, "", nil)
}

func talk() {
	if *ak == "" || *sk == "" || *nsid == "" || *gbid == "" || *audioFile == "" {
		flag.PrintDefaults()
		return
	}
	/*
		audio, err := ioutil.ReadFile(*audioFile)
		if err != nil {
			log.Println(err)
			return
		}
		jsonBody := "{\"base64Audio\":\"" + string(audio[:len(audio)-1]) + "\"}"
	*/
	body := "{\"isV2\":true}"
	host := "qvs.qiniuapi.com"
	headers := map[string]string{"Content-Type": "application/json"}
	path := fmt.Sprintf("/v1/namespaces/%s/devices/%s/talk", *nsid, *gbid)
	resp, err := httpPost(*ak, *sk, host, path, body, headers)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(resp))
}

func pm3u8() {
	if *addr == "" {
		flag.PrintDefaults()
		return
	}
	resp, err := httpGet(*addr)
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
	addr = flag.String("url", "", "pm3u8 url")
	audioFile = flag.String("audiofile", "", "audio file")
	flag.Parse()
}

func qvsTestGet(path string) {
	addr := fmt.Sprintf("http://qvs-test.qiniuapi.com/v1/%s", path)
	resp, err := httpGet(addr)
	if err != nil {
		return
	}
	log.Println(resp)
}

func main() {
	log.SetFlags(log.Lshortfile)
	parseConsole()
	//pm3u8()
	qvsTestGet(fmt.Sprintf("namespaces/%s/baches", *nsid))
}

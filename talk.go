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
	for key, value := range headers {
		data += "\n" + key + ": " + value
	}
	data += "\n\n"
	if body != "" {
		data += body
	}
	token := "Qiniu " + ak + ":" + hmacSha1(sk, data)
	return token
}

func httpPost(ak, sk, host, path string, body []byte, headers map[string]string) ([]byte, error) {
	method := "POST"
	token := signToken(ak, sk, method, path, host, string(body), headers)
	client := &http.Client{}
	req, _ := http.NewRequest(method, "http://"+host+path, bytes.NewBuffer(body))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.Header.Add("Authorization", token)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	return resp_body, err
}

func main() {
	/*
		host := "qvs.qiniuapi.com"
		path := "/v1/namespaces/3nm4x0vyz7xlu/devices/31011500991320002638/start"
		ak := "JAwTPb8dmrbiwt89Eaxa4VsL4_xSIYJoJh4rQfOQ"
		sk := "G5mtjT3QzG4Lf7jpCAN5PZHrGeoSH9jRdC96ecYS"
		headers := map[string]string{"Content-Type": "application/json"}
		resp, err := httpPost(ak, sk, host, path, []byte(""), headers)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(resp))
	*/
	log.SetFlags(log.Lshortfile)
	ak := flag.String("ak", "", "ak")
	sk := flag.String("sk", "", "sk")
	nsid := flag.String("nsid", "", "namespace id")
	gbid := flag.String("gbid", "", "gbid")
	audioFile := flag.String("audiofile", "", "audio file")
	flag.Parse()
	if *ak == "" || *sk == "" || *nsid == "" || *gbid == "" || *audioFile == "" {
		flag.PrintDefaults()
		return
	}
	audio, err := ioutil.ReadFile(*audioFile)
	if err != nil {
		log.Println(err)
		return
	}
	jsonBody := "{\"base64Audio\":\"" + string(audio[:len(audio)-1]) + "\"}"
	host := "qvs-test.qiniuapi.com"
	headers := map[string]string{"Content-Type": "application/json"}
	path := fmt.Sprintf("/v1/namespaces/%s/devices/%s/talk", *nsid, *gbid)
	resp, err := httpPost(*ak, *sk, host, path, []byte(jsonBody), headers)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(resp))
}

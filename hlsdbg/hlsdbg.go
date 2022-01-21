package main

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"hlsdbg/m3u8"
)

var (
	errHttpStatusCode = errors.New("http status code err")
)

var addr *string

func parseConsole() {
	addr = flag.String("url", "", "hls播放地址")
	flag.Parse()
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

func main() {
	log.SetFlags(log.Lshortfile)
	m3u8.Test()
}

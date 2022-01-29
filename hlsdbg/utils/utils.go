package utils

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var (
	errHttpStatusCode = errors.New("http status code err")
)

func HttpReq(method, addr, body string, headers map[string]string) (string, error) {
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

func HttpGet(addr string) (string, int64, error) {
	start := time.Now().UnixMilli()
	resp, err := HttpReq("GET", addr, "", nil)
	cost := time.Now().UnixMilli() - start
	return resp, cost, err
}

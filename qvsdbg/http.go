package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

func httpReq(method, addr, body string, headers map[string]string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http client do err: %v", err)
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("http client read body err: %v", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("http resp code err, statusCode: %d status: %s", resp.StatusCode, resp.Status)
	}
	return string(resp_body), nil
}

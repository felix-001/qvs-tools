package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/qbox/bo-sdk/base/xlog.v1"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"
)

var (
	errHttpStatusCode = errors.New("http status code err")
)

func (s *Parser) post(addr, jsonData string, out any) error {
	req, err := http.NewRequest("POST", addr, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	// 设置请求头，指定发送的数据是 JSON 格式
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// 读取响应数据
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		log.Println(err, string(body))
		return err
	}
	return nil
}

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
	log.Println("data:")
	fmt.Println(data)
	token := "Qiniu " + ak + ":" + hmacSha1(sk, data)
	log.Println("token:", token)
	return token
}

func httpReq(method, addr, body string, headers map[string]string) (string, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	log.Printf("%+v\n", req)
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
		log.Println("status code", resp.StatusCode, string(resp_body), resp.Status)
		return "", errHttpStatusCode
	}
	return string(resp_body), err
}

func mikuHttpReq(method, addr, body, ak, sk string) (string, error) {
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

func (s *Parser) s3get(addr string) (string, error) {
	qc := qconfapi.New(&s.conf.AccountCfg)
	ag := appg.Client{Conn: qc}
	uid, err := strconv.Atoi(s.conf.Uid)
	if err != nil {
		log.Fatalln(err)
	}
	ak, sk, err := ag.GetAkSk(xlog.FromContextSafe(context.Background()), uint32(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return mikuHttpReq("GET", addr, "", ak, sk)
}

func (s *Parser) s3patch(addr, body string) (string, error) {
	qc := qconfapi.New(&s.conf.AccountCfg)
	ag := appg.Client{Conn: qc}
	uid, err := strconv.Atoi(s.conf.Uid)
	if err != nil {
		log.Fatalln(err)
	}
	ak, sk, err := ag.GetAkSk(xlog.FromContextSafe(context.Background()), uint32(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return mikuHttpReq("PATCH", addr, body, ak, sk)
}

func get(addr string) (string, error) {
	return httpReq("GET", addr, "", nil)
}

func getWithBody(addr, body string) (string, error) {
	return httpReq("GET", addr, body, nil)
}

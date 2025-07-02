package util

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
	"mikutool/config"
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

func Post(addr, jsonData string, out any) error {
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
	//log.Println("data:")
	//fmt.Println(data)
	token := "Qiniu " + ak + ":" + hmacSha1(sk, data)
	//log.Println("token:", token)
	return token
}

func HttpReq(method, addr, body string, headers map[string]string) (string, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		if key == "Host" {
			req.Host = value
			continue
		}
		req.Header.Add(key, value)
	}
	//log.Printf("%+v\n", req)
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

func QnHttpReq(method, addr, body, ak, sk string) (string, error) {
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
	return HttpReq(method, addr, body, headers)
}

func S3get(addr string, conf *config.Config) (string, error) {
	qc := qconfapi.New(&conf.AccountCfg)
	ag := appg.Client{Conn: qc}
	uid, err := strconv.Atoi(conf.Uid)
	if err != nil {
		log.Fatalln(err)
	}
	ak, sk, err := ag.GetAkSk(xlog.FromContextSafe(context.Background()), uint32(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return QnHttpReq("GET", addr, "", ak, sk)
}

func S3patch(addr, body string, conf *config.Config) (string, error) {
	qc := qconfapi.New(&conf.AccountCfg)
	ag := appg.Client{Conn: qc}
	uid, err := strconv.Atoi(conf.Uid)
	if err != nil {
		log.Fatalln(err)
	}
	ak, sk, err := ag.GetAkSk(xlog.FromContextSafe(context.Background()), uint32(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return QnHttpReq("PATCH", addr, body, ak, sk)
}

func Get(addr string) (string, error) {
	return HttpReq("GET", addr, "", nil)
}

func GetWithBody(addr, body string) (string, error) {
	return HttpReq("GET", addr, body, nil)
}

func Http(conf *config.Config) {
	method := "GET"
	if conf.Body != "" {
		if conf.Method != "" {
			method = conf.Method
		} else {
			method = "POST"
		}
	}
	if conf.Addr == "" {
		log.Println("need -addr <url>")
		return
	}
	if conf.Ak == "" {
		log.Println("need -ak <ak>")
		return
	}
	if conf.Sk == "" {
		log.Println("need -sk <sk>")
		return
	}
	resp, err := QnHttpReq(method, conf.Addr, conf.Body, conf.Ak, conf.Sk)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(resp)
}

func httpReqReturnHdr(method, addr, body string, headers map[string]string) (int, string, http.Header) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		if key == "Host" {
			req.Host = value
			continue
		}
		req.Header.Add(key, value)
	}
	//log.Printf("%+v\n", req)
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return 0, err.Error(), nil
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	//log.Print("resp body", string(resp_body))
	if err != nil {
		log.Println(err)
		return 0, err.Error(), nil
	}
	return resp.StatusCode, string(resp_body), resp.Header
}

func QnHttpReqReturnHdr(method, addr, body, ak, sk string) (int, string, http.Header) {
	u, err := url.Parse(addr)
	if err != nil {
		log.Println(err)
		return 0, err.Error(), nil
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
	return httpReqReturnHdr(method, addr, body, headers)
}

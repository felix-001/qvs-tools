package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
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

var (
	ak     string
	sk     string
	path   string
	body   string
	host   string
	defAK  string
	defSK  string
	method string
)

var (
	errHttpStatusCode = errors.New("http status code err")
)

const (
	apiHost     = "http://qvs.qiniuapi.com"
	apiHostTest = "http://qvs-test.qiniuapi.com"
	conf        = "/usr/local/etc/qvsHttpCli.conf"
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
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}
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
	//log.Print("resp body", string(resp_body))
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

func qvsHttpReq(method, addr, body string) (string, error) {
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

func parseConf() {
	b, err := ioutil.ReadFile(conf)
	if err != nil {
		log.Printf("%s not found\n", conf)
		return
	}
	keys := struct {
		AK string `json:"ak"`
		SK string `json:"sk"`
	}{}
	if err := json.Unmarshal(b, &keys); err != nil {
		log.Println("parse conf err", err)
		return
	}
	defAK = keys.AK
	defSK = keys.SK
}

func parseConsole() {
	flag.StringVar(&ak, "ak", defAK, "ak")
	flag.StringVar(&sk, "sk", defSK, "sk")
	flag.StringVar(&path, "path", "", "path")
	flag.StringVar(&body, "body", "", "body")
	flag.StringVar(&host, "host", apiHost, "host")
	flag.StringVar(&method, "method", "GET", "method")
	flag.Parse()
	if ak == "" {
		log.Println("need ak")
		flag.PrintDefaults()
		os.Exit(0)
	}
	if sk == "" {
		log.Println("need sk")
		flag.PrintDefaults()
		os.Exit(0)
	}
	if path == "" {
		log.Println("need path")
		flag.PrintDefaults()
		os.Exit(0)
	}
}

func isJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func PrintJSON(m map[string]interface{}) {
	for k, v := range m {
		switch vv := v.(type) {
		case string:
			fmt.Println(k, "is string", vv)
		case float64:
			fmt.Println(k, "is float", int64(vv))
		case int:
			fmt.Println(k, "is int", vv)
		case []interface{}:
			fmt.Println(k, "is an array:")
			for i, u := range vv {
				fmt.Println(i, u)
			}
		case nil:
			fmt.Println(k, "is nil", "null")
		case bool:
			fmt.Println(k, "is bool", vv)
		case map[string]interface{}:
			fmt.Println(k, "is an map:")
			PrintJSON(vv)
		default:
			fmt.Println(k, "is of a type I don't know how to handle ", fmt.Sprintf("%T", v))
		}
	}
}

func printJson1(data string) string {
	var d interface{}
	err := json.Unmarshal([]byte(data), &d)
	if err != nil {
		fmt.Println("JSON unmarshaling failed:", err)
		return ""
	}
	jsonStr, err := json.MarshalIndent(d, "", "   ")
	if err != nil {
		fmt.Println("JSON marshaling failed:", err)
		return ""
	}
	return string(jsonStr)
}

func main() {
	log.SetFlags(log.Lshortfile)
	parseConf()
	parseConsole()
	uri := fmt.Sprintf("%s%s", host, path)
	if method == "" {
		method = "GET"
		if body != "" {
			method = "POST"
		}
	}
	resp, err := qvsHttpReq(method, uri, body)
	if err != nil {
		log.Println("err:", err)
		return
	}
	resp = strings.ReplaceAll(resp, "\u0026", "")

	/*
		if isJSON(resp) {
			m := map[string]interface{}{}
			if err := json.Unmarshal([]byte(resp), &m); err != nil {
				log.Fatal("err:", err)
				return
			}
			PrintJSON(m)
				for key, value := range m {

					log.Println(key, ":", value, reflect.TypeOf(value))
				}

			return
		}
	*/
	log.Println("resp", printJson1(resp))
}

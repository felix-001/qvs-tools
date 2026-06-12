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
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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
	if body != "" {
		data += "\n" + "Content-Type" + ": " + "application/json"
	}
	for key, value := range headers {
		if key != "Content-Type" {
			continue
		}
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

func HttpReq(method, addr, body string, headers map[string]string, detailLog bool) (string, error) {
	dailCtx := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := net.Dialer{Timeout: 120 * time.Second}
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		// 获取并打印服务端地址
		/*
			if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
				fmt.Printf("Connected to server IP: %s\n", tcpAddr.IP.String())
			} else {
				fmt.Printf("Connected to server: %s\n", conn.RemoteAddr().String())
			}
		*/

		return conn, nil
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, DialContext: dailCtx}

	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		if key == "Host" {
			req.Host = value
			continue
		}
		req.Header.Add(key, value)
	}
	//req.Header.Set("Connection", "close")
	//req.Header.Set("X-Provider", "portal")
	if method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")

	}
	if detailLog {
		log.Printf("%+v\n", req)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	//fmt.Printf("resp: %+v\n", resp)
	//log.Print("resp body", string(resp_body))
	if err != nil {
		log.Println(err)
		return "", err
	}
	if detailLog {
		log.Printf("resp header: %+v\n", resp.Header)
	}
	if resp.StatusCode != 200 {
		log.Println("status code:", resp.StatusCode, "body:", string(resp_body), "status:", resp.Status)
		return "", errHttpStatusCode
	}
	return string(resp_body), err
}

func QnHttpReq(method, addr, body, ak, sk string, headerMap map[string]string, detailLog bool) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		log.Println(err)
		return "", err
	}
	host := u.Host
	u.Host = ""
	u.Scheme = ""
	headers := headerMap
	/*
		if body != "" || method == "PUT" {
			headers["Content-Type"] = "application/json"
		}
	*/
	//token := signToken(ak, sk, method, u.Path, u.Host, body, headers)
	token := signToken(ak, sk, method, u.String(), host, body, headers)
	headers["Authorization"] = token
	return HttpReq(method, addr, body, headers, detailLog)
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
	return QnHttpReq("GET", addr, "", ak, sk, map[string]string{}, conf.Detail)
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
	return QnHttpReq("PATCH", addr, body, ak, sk, map[string]string{}, conf.Detail)
}

func Get(addr string, detailLog bool) (string, error) {
	return HttpReq("GET", addr, "", nil, detailLog)
}

func GetWithBody(addr, body string, detailLog bool) (string, error) {
	return HttpReq("GET", addr, body, nil, detailLog)
}

func Http(conf *config.Config) (string, error) {
	method := "GET"
	if conf.Method != "" {
		method = conf.Method
	}
	if conf.Body != "" && conf.Method == "" {
		method = "POST"
	}
	if conf.Uid != "" {
		conf.Ak, conf.Sk = GetAkSk(conf)
	}
	if conf.User != "" {
		bytes, err := os.ReadFile(fmt.Sprintf("/usr/local/etc/%s.txt", conf.User))
		if err != nil {
			log.Println(err)
			return "", err
		}
		ss := strings.Split(string(bytes)[0:len(bytes)-1], ",")
		if len(ss) != 2 {
			log.Println("user file is empty")
			return "", fmt.Errorf("user file is empty")
		}
		conf.Ak = ss[0]
		conf.Sk = ss[1]
		if conf.Detail {
			log.Println("ak:", conf.Ak, "sk:", conf.Sk)
		}
	}
	if conf.Path != "" {
		domain := "mls-test.cn-east-1.qiniumiku.com"
		switch conf.User {
		case "gray", "qa":
			domain = "mls-test.cn-east-1.qiniumiku.com"
		case "mikutest":
			// uid: 1381218095
			domain = "mls.cn-east-1.jfcs.qiniu.io"
		case "mikuonline", "qvsmiku":
			domain = "mls.cn-east-1.qiniumiku.com"
		case "qvs":
			domain = "qiniuapi.com"
		}
		if conf.Bucket == "" {
			log.Println("need -bucket <bucket>")
			return "", fmt.Errorf("missing bucket")
		}
		switch conf.Path {
		case "domain":
			if conf.Domain == "" {
				log.Println("need -domain <domain>")
				return "", fmt.Errorf("missing domain")
			}
			conf.Addr = fmt.Sprintf("http://%s.%s/?domainConfig&name=%s", conf.Bucket, domain, conf.Domain)
		case "bucket":
			conf.Addr = fmt.Sprintf("http://%s.%s/?config", conf.Bucket, domain)
		case "roominfo":
			if conf.ID == "" {
				log.Println("need -id")
				return "", fmt.Errorf("need -id")
			}
			conf.Addr = fmt.Sprintf("http://%s.%s/?roomrti&roomid=%s", conf.Bucket, domain, conf.ID)
		case "listroom":
			conf.Addr = fmt.Sprintf("http://%s.%s/?roomrtis&offset=%d&limit=%d", conf.Bucket, domain, conf.Offset, conf.Limit)
		case "userinfo":
			if conf.ID == "" {
				log.Println("need -id")
				return "", fmt.Errorf("need -id")
			}
			if conf.Uid == "" {
				log.Println("need -uid")
				return "", fmt.Errorf("need -uid")
			}
			conf.Addr = fmt.Sprintf("http://%s.%s/?userrti&roomid=%s&userid=%s", conf.Bucket, domain, conf.ID, conf.Uid)
		case "listuser":
			if conf.ID == "" {
				log.Println("need -id")
				return "", fmt.Errorf("need -id")
			}
			conf.Addr = fmt.Sprintf("http://%s.%s/?userrtis&roomid=%s&offset=%d&limit=%d", conf.Bucket, domain, conf.ID, conf.Offset, conf.Limit)
		case "deleteuser":
			method = "DELETE"
			if conf.ID == "" {
				log.Println("need -id")
				return "", fmt.Errorf("need -id")
			}
			if conf.Uid == "" {
				log.Println("need -uid")
				return "", fmt.Errorf("need -uid")
			}
			conf.Addr = fmt.Sprintf("http://%s.%s/?userrti&roomid=%s&userid=%s", conf.Bucket, domain, conf.ID, conf.Uid)
		case "deleteroom":
			method = "DELETE"
			if conf.ID == "" {
				log.Println("need -id")
				return "", fmt.Errorf("need -id")
			}
			conf.Addr = fmt.Sprintf("http://%s.%s/?roomrti&roomid=%s", conf.Bucket, domain, conf.ID)
		case "authrti":
			conf.Addr = fmt.Sprintf("http://%s.%s/?authrti", conf.Bucket, domain)
		case "wm": // watermark
			conf.Addr = fmt.Sprintf("http://%s/?watermarkTemplate", domain)
			if conf.Method == "PATCH" || method == "GET" || method == "DELETE" {
				conf.Addr += fmt.Sprintf("&id=%s", conf.ID)
			}
		case "upwm": // upload watermark
			conf.Addr = fmt.Sprintf("http://%s/?watermarkImageUpload", domain)
			if method == "DELETE" {
				conf.Addr += fmt.Sprintf("&fileName=%s", conf.Name)
			}
		case "listwm": // list watermark
			conf.Addr = fmt.Sprintf("http://%s/?watermarkTemplates", domain)
		case "codec": // update codec template
			conf.Addr = fmt.Sprintf("http://%s/?codecTemplate", domain)
		default:
			conf.Addr = fmt.Sprintf("http://%s", domain)
		}
	}
	if conf.Addr == "" {
		log.Println("need -addr <url>")
		return "", fmt.Errorf("missing address")
	}
	if conf.Ak == "" {
		log.Println("need -ak <ak>")
		return "", fmt.Errorf("missing ak")
	}
	if conf.Sk == "" {
		log.Println("need -sk <sk>")
		return "", fmt.Errorf("missing sk")
	}
	if conf.Body != "" {
		conf.HeaderMap["content-type"] = "application/json"
	}
	log.Println("headers:", conf.HeaderMap)

	resp, err := QnHttpReq(method, conf.Addr, conf.Body, conf.Ak, conf.Sk, conf.HeaderMap, conf.Detail)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if conf.Detail {
		fmt.Println("raw resp:", resp)
	}
	respMap := make(map[string]any)
	if err := json.Unmarshal([]byte(resp), &respMap); err != nil {
		log.Println("err:", err)
		return "", err
	}
	bytes, err := json.MarshalIndent(respMap, "", "  ")
	if err != nil {
		log.Println(err)
		return "", err
	}
	fmt.Println(string(bytes))
	return string(bytes), err
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

func HttpRequest(conf *config.Config) (string, error) {
	method := "GET"
	if conf.Method != "" {
		method = conf.Method
	}
	if conf.Body != "" && conf.Method == "" {
		method = "POST"
	}
	if conf.Uid != "" {
		conf.Ak, conf.Sk = GetAkSk(conf)
	}
	if conf.Addr == "" {
		log.Println("need -addr <url>")
		return "", fmt.Errorf("missing address")
	}
	if conf.Ak == "" {
		log.Println("need -ak <ak>")
		return "", fmt.Errorf("missing ak")
	}
	if conf.Sk == "" {
		log.Println("need -sk <sk>")
		return "", fmt.Errorf("missing sk")
	}
	//log.Println("headers:", conf.HeaderMap)

	resp, err := QnHttpReq(method, conf.Addr, conf.Body, conf.Ak, conf.Sk, conf.HeaderMap, conf.Detail)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return resp, nil
}

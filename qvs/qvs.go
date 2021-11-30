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
var gbids *string

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
	if resp.StatusCode != 200 {
		log.Println("not 200")
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

func qvsTestPost(path, body string) {
	addr := fmt.Sprintf("http://qvs-test.qiniuapi.com/v1/%s", path)
	resp, err := qvsHttpPost(addr, body)
	if err != nil {
		return
	}
	log.Println(resp)

}

const BlkLen = 5 * 1024

func calcBlkLen(len, pos int) int {
	blkLen := BlkLen
	left := len - pos
	if left < BlkLen {
		blkLen = left
	}
	return blkLen
}

func sendBlk(blk []byte, addr string) error {
	base64Blk := base64.RawURLEncoding.EncodeToString(blk)
	body := fmt.Sprintf("{\"append_audio_pcm\":%s}", base64Blk)
	resp, err := httpPost(addr, body)
	if err != nil {
		return err
	}
	log.Println(resp)
	return nil
}

func broadcast() {
	qvsTestPost(fmt.Sprintf("namespaces/%s/talks", *nsid))
	pcm, err := ioutil.ReadFile(*audioFile)
	if err != nil {
		log.Println(err)
		return
	}
	pos := 0
	for pos < len(pcm) {
		blkLen := calcBlkLen(len(pcm), pos)
		blk := pcm[pos : pos+blkLen]
		err := sendBlk(blk)
		if err != nil {
			return
		}
		pos += BlkLen
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	parseConsole()
	qvsTestGet(fmt.Sprintf("namespaces/%s/baches", *nsid))
}

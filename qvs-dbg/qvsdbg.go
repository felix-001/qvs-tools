package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
		log.Println("status code", resp.StatusCode, string(resp_body), addr)
		return "", errors.New("http resp code err")
	}
	return string(resp_body), err
}

func qvsReq(method, addr, body, uid string) (string, error) {
	headers := map[string]string{"authorization": "QiniuStub uid=0"}
	return httpReq(method, addr, body, headers)
}

/*
curl http://{adminIP}:7277/v1/namespaces --header 'authorization: QiniuStub uauthorization: QiniuStub uid=xxx'
*/
func getSIPNode(adminIP, gbid) (string, error) {
	uri := fmt.Sprintf("http://%s:7277/v1/v1/namespaces/devices/%s", adminIP, gbid)
	resp, err := qvsReq("GET", uri, "")
	if err != nil {
		return "", err
	}
}

func getSSRC(reqId, adminIP, gbid string) (string, error) {
}

/*
1. 拉流失败
	1.1 实时流
	1.2 历史流
2. 设备离线
3. 对讲失败
4. 云端录制失败
*/

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	reqId := flag.String("reqId", "reqid", "input reqid")
	adminIP := flag.String("adminip", "", "input admin ip")
	gbid := flag.String("gbid", "", "input gbid")
	flag.Parse()
	if *reqId == "" {
		flag.PrintDefaults()
		return
	}
	ssrc := getSSRC(*reqId, *adminIP, *gbid)
	log.Println(ssrc)
}

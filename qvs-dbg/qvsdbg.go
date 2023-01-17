package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var (
	adminIP string
	gbID    string
	reqID   string
)

type Context struct {
	NsId      string
	SipNodeID string
}

func NewContext() *Context {
	return &Context{}
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
		log.Println("status code", resp.StatusCode, string(resp_body), addr)
		return "", errors.New("http resp code err")
	}
	return string(resp_body), err
}

func qvsReq(method, addr, body string) (string, error) {
	headers := map[string]string{"authorization": "QiniuStub uid=0"}
	return httpReq(method, addr, body, headers)
}

func qvsGet(addr string) (string, error) {
	return qvsReq("GET", addr, "")
}

type Device struct {
	Vendor   string `json:"vendor"`
	RemoteIP string `json:"remoteIp"`
	NsID     string `json:"nsId"`
	NodeID   string `json:"nodeId"`
}

func (c *Context) getNsID() (string, error) {
	path := fmt.Sprintf("devices?prefix=%s", gbID)
	uri := c.adminUri(path)
	resp, err := qvsGet(uri)
	if err != nil {
		return "", err
	}
	//log.Println(resp)
	data := struct {
		Devices []Device `json:"items"`
	}{}
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		log.Println(err)
		return "", err
	}
	if len(data.Devices) == 0 {
		return "", fmt.Errorf("device not found")
	}
	c.NsId = data.Devices[0].NsID
	log.Println("nsId:", c.NsId)
	return data.Devices[0].NsID, nil
}

func (c *Context) adminUri(path string) string {
	return fmt.Sprintf("http://%s:7277/v1/%s", adminIP, path)
}

func (c *Context) getSipNodeID() (string, error) {
	path := fmt.Sprintf("namespaces/%s/devices/%s", c.NsId, gbID)
	uri := c.adminUri(path)
	resp, err := qvsGet(uri)
	if err != nil {
		return "", err
	}
	var device Device
	if err := json.Unmarshal([]byte(resp), &device); err != nil {
		log.Println(err)
		return "", err
	}
	sipNodeId := device.NodeID
	ss := strings.Split(device.NodeID, "_")
	if len(ss) == 2 {
		sipNodeId = ss[0]
	}
	c.SipNodeID = sipNodeId
	log.Println("sip nodeid:", c.SipNodeID)
	return device.NodeID, nil
}

func (c *Context) getSSRC(reqId string) (string, error) {
	return "", nil
}

func parseConsole() error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.StringVar(&reqID, "r", "", "input reqid")
	flag.StringVar(&adminIP, "i", "", "input admin ip")
	flag.StringVar(&gbID, "g", "", "input gbid")
	flag.Parse()
	if reqID == "" {
		return fmt.Errorf("please input reqId")
	}
	if adminIP == "" {
		return fmt.Errorf("please input adminip")
	}
	if gbID == "" {
		return fmt.Errorf("please input gbid")
	}
	return nil
}

/*
1. 拉流失败
	1.1 实时流
	1.2 历史流
2. 设备离线
3. 对讲失败
4. 云端录制失败
5. 如果通过admin获取不到nodeId，则通过pdr去获取
^[[31m[2023-01-17 11:33:57.440][Error][56736][d4262t5f][11] ssrc illegal on tcp payload chaanellid: ssrc :123116795 rtp_pack_len:8012
buf:13142 payload_type:96 peer_ip:120.193.152.166:26564 fd:39(Resource temporarily unavailable)
6. ssrc不正确的问题
*/

func main() {
	if err := parseConsole(); err != nil {
		log.Println(err)
		flag.PrintDefaults()
		return
	}
	//ssrc := getSSRC()
	//log.Println(ssrc)
	ctx := NewContext()
	if _, err := ctx.getNsID(); err != nil {
		log.Println(err)
		return
	}
	if _, err := ctx.getSipNodeID(); err != nil {
		log.Println(err)
		return
	}
}

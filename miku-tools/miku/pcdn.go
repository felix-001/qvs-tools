package miku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"mikutool/miku/users"
	"mikutool/public/util"
	"net/http"
	"strconv"
	"time"
)

type HuyaP2pPcdnBatchPostBody struct {
	StreamNameArr []string `json:"streamname_arr"`
}

func (m *Miku) GetPCDN() {
	for i := 0; i < 30; i++ {
		go m.loopReq()
	}
	time.Sleep(time.Second)
}

func (m *Miku) loopReq() {
	for i := 0; i < m.conf.N; i++ {
		resp, err := m.ExecuteGetPCDNBatchRequest()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Response:", string(resp))
	}

}

// ExecuteGetPCDNBatchRequest 执行请求 http://127.0.0.1:9090/huyalive/aaa/getpcdnbatch
func (m *Miku) ExecuteGetPCDNBatchRequest() ([]byte, error) {
	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 生成1-1000的随机数
	randomNum := rand.Intn(10) + 1

	// 将随机数转成字符串并格式化
	randomNumStr := strconv.Itoa(randomNum)
	if len(randomNumStr) == 1 {
		randomNumStr = "00" + randomNumStr
	} else if len(randomNumStr) == 2 {
		randomNumStr = "0" + randomNumStr
	}

	body := HuyaP2pPcdnBatchPostBody{
		StreamNameArr: []string{},
	}

	// 定义26个英文字母
	letters := "abcdef"
	for i := 0; i < 3; i++ {
		// 初始化随机字符串
		randomStr := "test"
		// 随机挑选1个字母
		randomStr += string(letters[rand.Intn(len(letters))])
		body.StreamNameArr = append(body.StreamNameArr, randomStr)
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	fmt.Println("jsonBody:", string(jsonBody))
	t, err := util.Str2time("2034-12-01 00:00:00")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	wsTime := fmt.Sprintf("%x", t.Unix())
	wsSecret := users.HuyaP2pToken("teststream", wsTime)
	addr := fmt.Sprintf("http://127.0.0.1:9090/huyalive/teststream/getpcdnbatch?uid=%s&wsTime=%s&wsSecret=%s&ClientIp=%s",
		randomNumStr, wsTime, wsSecret, m.conf.Ip)
	resp, err := http.Post(addr, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

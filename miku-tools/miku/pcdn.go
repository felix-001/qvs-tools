package miku

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
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

	zerolog "github.com/rs/zerolog/log"
)

type HuyaP2pPcdnBatchPostBody struct {
	StreamNameArr []string `json:"streamname_arr"`
}

func (m *Miku) GetPCDN() {
	if m.conf.Bucket == "dy" {
		m.GetDouyuPCDN()
		return
	}
	for i := 0; i < m.conf.Loop; i++ {
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
	randomNum := rand.Intn(22) + 1

	// 将随机数转成字符串并格式化
	randomNumStr := strconv.Itoa(randomNum)
	if len(randomNumStr) == 1 {
		randomNumStr = "uid-new-00" + randomNumStr
	} else if len(randomNumStr) == 2 {
		randomNumStr = "uid-new-0" + randomNumStr
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
	addr := fmt.Sprintf("http://%s:9090/huyalive/teststream/getpcdnbatch?uid=%s&wsTime=%s&wsSecret=%s&ClientIp=%s",
		m.conf.SchedIp, randomNumStr, wsTime, wsSecret, m.conf.Ip)
	resp, err := http.Post(addr, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func DouyuAuth(streamName, wsTime, secret string) string {
	raw := secret + streamName + wsTime
	hash := md5.Sum([]byte(raw))
	expectedSecret := hex.EncodeToString(hash[:])
	zerolog.Info().Str("streamName", streamName).
		Str("wsTime", wsTime).
		Str("expectedSecret", expectedSecret).
		Str("raw", raw).
		Msg("[DouyuAuth]")
	return expectedSecret
}

func (m *Miku) GetDouyuPCDN() {
	if m.conf.Secret == "" {
		log.Println("secret is empty")
		return
	}
	t := time.Now().Unix() + 600
	tHex := strconv.FormatInt(t, 16)
	wsSecret := DouyuAuth(m.conf.Stream, tHex, m.conf.Secret)
	addr := fmt.Sprintf("http://miku-lived-test.qiniuapi.com/live/%s.xs?wsTime=%s&wsSecret=%s", m.conf.Stream, tHex, wsSecret)
	resp, err := http.Get(addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(body))

}

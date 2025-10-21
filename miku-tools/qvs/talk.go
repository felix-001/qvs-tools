package qvs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"mikutool/public/util"
	"net/http"
)

type TalkResp struct {
	AudioSendAddrForHttp  string `json:"audioSendAddrForHttp"`
	AudioSendAddrForHttps string `json:"audioSendAddrForHttps"`
}

func TalkApi(conf *config.Config) {
	if conf.Uid != "" {
		conf.Ak, conf.Sk = util.GetAkSk(conf)
	} else {
		log.Println("need uid")
		return
	}
	addr := fmt.Sprintf("http://qvs.qiniuapi.com/v1/namespaces/%s/devices/%s/talk", conf.NsId, conf.GBId)
	body := `{ "isV2": true, "version": "2014", "transProtocol":"tcp" }`
	respStr, err := util.QnHttpReq("POST", addr, body, conf.Ak, conf.Sk, map[string]string{})
	if err != nil {
		log.Println("http req err", err)
		return
	}
	resp := &TalkResp{}
	if err := json.Unmarshal([]byte(respStr), resp); err != nil {
		log.Println("unmarshal err", err, respStr)
		return
	}
	log.Println("talk api success, audioSendAddrForHttp:", resp.AudioSendAddrForHttp)
	log.Println("talk api success, audioSendAddrForHttps:", resp.AudioSendAddrForHttps)
	//time.Sleep(1 * time.Second)
	buf := Zero102400()
	voice := base64.StdEncoding.EncodeToString(buf)
	if !conf.Silence {
		log.Println("use test voice")
		voice = TalkTestVoice
	}
	respStr, err = AppendAudioPCM(resp.AudioSendAddrForHttp, voice)
	if err != nil {
		log.Println("append audio pcm err", err, resp)
		return
	}
	log.Println("append audio pcm success:", respStr)

}

func AppendAudioPCM(addr, base64PCM string) (string, error) {
	// 构造带查询参数的 URL
	url := addr

	// 构造 JSON body
	payload := map[string]string{
		"base64_pcm": base64PCM,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// 发送 POST 请求
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return string(respBody), fmt.Errorf("http status %s", resp.Status)
	}
	return string(respBody), nil
}

func Zero102400() []byte {
	buf := make([]byte, 102410)
	// make 初始化即为 0x00，下面的循环是显式设置（可选）
	for i := range buf {
		buf[i] = 0xd5
	}
	return buf
}

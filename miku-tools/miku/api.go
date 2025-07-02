package miku

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"
)

type PlaycheckReq struct {
	Bucket   string            `json:"bucket"`
	Key      string            `json:"key"`
	Url      string            `json:"url"`
	Remote   string            `json:"remoteAddr"`
	Local    string            `json:"localAddr"`
	Node     string            `json:"nodeId"`
	ConnId   string            `json:"connectId"`
	Master   string            `json:"master"`
	Protocol string            `json:"protocol"` // 协议类型，非必填，目前仅hls协议传递
	Headers  map[string]string `json:"headers"`
	User     string            `json:"user"`
}

type PlayCheckResp struct {
	Uid        uint32 `json:"uid"`
	ErrCode    string `json:"code"`
	Message    string `json:"message"`
	ConnectId  string `json:"connectId"`
	Url302     string `json:"redirectUrl"`
	RewriteUrl string `json:"rewriteUrl"`
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	RegTsM3u8  string `json:"regTsM3u8"`  // m3u8文件中ts格式，替换其中的${app}/${stream}/${file}
	FlowMethod int    `json:"flowMethod"` // 计量方式: 1: miku计量系统; 2: pili计量系统; 其它值miku&pili计量系统
}

func playcheck(ip string, conf *config.Config) *PlayCheckResp {
	scheme := "http"
	if conf.Https {
		scheme += "s"
	}
	playUrl := fmt.Sprintf("%s://%s/%s/%s%s?did=a75e6982-7538-4629-ad3c-fd0d60b1ba54&expire=0",
		scheme, conf.Domain, conf.Bucket, conf.Stream, conf.Format)
	req := PlaycheckReq{
		Bucket: conf.Bucket,
		Key:    conf.Stream,
		Url:    playUrl,
		Node:   conf.Node,
		Remote: ip,
		ConnId: conf.ConnId,
		User:   conf.User,
	}
	bytes, err := json.Marshal(&req)
	if err != nil {
		log.Println(err)
		return nil
	}
	var resp PlayCheckResp
	addr := fmt.Sprintf("http://%s:6060/api/v1/playcheck", conf.SchedIp)
	if err := util.Post(addr, string(bytes), &resp); err != nil {
		log.Println(err)
		return nil
	}
	return &resp
}

func Playcheck(conf *config.Config) {
	resp := playcheck(conf.SchedIp+":8080", conf)
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

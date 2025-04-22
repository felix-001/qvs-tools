package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	node1IP     = "172.18.0.2"
	node2IP     = "172.18.0.3"
	node3IP     = "172.18.0.4"
	idc         = "vdn"
	callbackUrl = "http://host.docker.internal:8090/callback"
)

type streamPublishCheckArgs struct {
	ID string `json:"id"`
}

type HubInfo struct {
	Name                  string  `json:"name"`
	HlsMinus              bool    `json:"hlsMinus"`
	HlsFileDuration       float64 `json:"hlsFileDuration"`
	HlsFileCount          int     `json:"hlsFileCount"`
	HlsMinFileCount       int     `json:"hlsMinFileCount"`
	StopStreamAfterExpire bool    `json:"stopStreamAfterExpire"`
	Callback              string  `json:"callback"`
	RecordTemplateId      string  `json:"recordTemplateId"`
}

type publishCheckResp struct {
	StreamId string  `json:"streamId"`
	HubInfo  HubInfo `json:"hubInfo"`
}

func streamPublishCheck(w http.ResponseWriter, req *http.Request) {
	log.Println("stream publish check")
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
		return
	}
	var args streamPublishCheckArgs
	if err := json.Unmarshal(body, &args); err != nil {
		log.Println(err)
		return
	}
	ss := strings.Split(args.ID, "/")
	if len(ss) != 2 {
		log.Println("split", args.ID, "by '/' err")
		return
	}
	streamId := fmt.Sprintf("%s:%s", ss[0], args.ID)
	resp := publishCheckResp{
		StreamId: streamId,
		HubInfo:  HubInfo{Name: ss[0], RecordTemplateId: "default"},
	}
	jbody, err := json.Marshal(resp)
	if err != nil {
		log.Println(err)
		return
	}
	if _, err := fmt.Fprint(w, string(jbody)); err != nil {
		log.Println(err)
		return
	}
}

type streamPlayCheckArgs struct {
	URL string `json:"url"`
}

type UpstreamInfo struct {
	NodeID        string `json:"node"`
	LocalIP       string `json:"localIP"`
	RemoteIP      string `json:"remoteIP"`
	RemoteNodeId  string `json:"remoteNodeId"`
	RemoteIdc     string `json:"remoteIdc"`
	RemoteDepth   int    `json:"remoteDepth"`
	RemoteURL     string `json:"remoteURL"`
	AbTestTag     string `json:"abTestTag"`
	TryPuicSocks5 bool   `json:"tryPuicSocks5"`
}

type playCheckResp struct {
	StreamId    string         `json:"streamId"`
	HubInfo     HubInfo        `json:"hubInfo"`
	TTL         int            `json:"ttl"`
	SourceInfos []UpstreamInfo `json:"sourceInfos"`
}

func streamPlayCheck(w http.ResponseWriter, req *http.Request) {
	log.Println("stream play check")
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
		return
	}
	var args streamPlayCheckArgs
	if err := json.Unmarshal(body, &args); err != nil {
		log.Println(err)
		return
	}
	u, err := url.Parse(args.URL)
	if err != nil {
		return
	}
	ss := strings.Split(u.Path, "/")
	if len(ss) != 3 {
		log.Println("split", u.Path, "err", len(ss))
		return
	}
	streamId := fmt.Sprintf("%s:%s/%s", ss[1], ss[1], ss[2])
	streamId = strings.ReplaceAll(streamId, ".m3u8", "")
	streamId = strings.ReplaceAll(streamId, ".flv", "")
	streamId = strings.ReplaceAll(streamId, ".wsflv", "")
	b64StreamId := base64.StdEncoding.EncodeToString([]byte(streamId))
	resp := playCheckResp{
		StreamId: streamId,
		HubInfo: HubInfo{
			HlsMinus:              true,
			HlsFileDuration:       2.0,
			HlsFileCount:          2,
			HlsMinFileCount:       2,
			StopStreamAfterExpire: true,
			Name:                  "gb28181",
			Callback:              callbackUrl,
		},
		SourceInfos: []UpstreamInfo{
			{
				LocalIP:       node3IP,
				RemoteIP:      node2IP,
				RemoteNodeId:  "node2",
				RemoteIdc:     idc,
				RemoteURL:     fmt.Sprintf("rtmp://%s/.i/%s", node2IP, b64StreamId),
				RemoteDepth:   0,
				AbTestTag:     "",
				TryPuicSocks5: false,
			},
			{
				LocalIP:       node2IP,
				RemoteIP:      node1IP,
				RemoteNodeId:  "node1",
				RemoteIdc:     idc,
				RemoteURL:     fmt.Sprintf("rtmp://%s/.i/%s", node1IP, b64StreamId),
				RemoteDepth:   1,
				AbTestTag:     "",
				TryPuicSocks5: false,
			},
		},
	}
	jbody, err := json.Marshal(resp)
	if err != nil {
		log.Println(err)
		return
	}
	if _, err := fmt.Fprint(w, string(jbody)); err != nil {
		log.Println(err)
		return
	}
}

func (s *Parser) mockThemisd() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	http.HandleFunc("/stream/publish/check", streamPublishCheck)
	http.HandleFunc("/stream/play/check", streamPlayCheck)
	http.ListenAndServe("0.0.0.0:6288", nil)
}

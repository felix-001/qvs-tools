package main

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

const RtpIp = "192.168.1.5"
const StreamUrl = "http://localhost:1985/api/v1/gb28181?action=talk&id=31010403001370002272&base64_pcm="
const InviteUrl = "http://localhost:7279/api/v1/gb28181?action=sip_invite&chid=31010403001370002272&id=31010403001370002272&ip=" + RtpIp + "&rtp_port=9001&rtp_proto=udp&is_talk=true&ssrc="
const ByeUrl = "http://localhost:7279/api/v1/gb28181?action=sip_bye&chid=31010403001370002272&id=31010403001370002272&&is_talk=true&call_id="
const BlkLen = 5 * 1024

type CreateStreamResp struct {
	Code int `json:"code"`
	Ssrc int `json:"ssrc"`
}

type InviteResp struct {
	Code   int    `json:"code"`
	Callid string `json:"call_id"`
}

func InviteAudio(ssrc int) string {
	if ssrc == 0 {
		log.Println("check ssrc error, ssrc is 0")
		return ""
	}
	inviteUrl := InviteUrl + strconv.Itoa(ssrc)
	resp, err := http.Get(inviteUrl)
	if err != nil {
		log.Println(err)
		return ""
	}
	log.Println("invite", resp.StatusCode)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return ""
	}
	inviteResp := &InviteResp{}
	err = json.Unmarshal(body, inviteResp)
	if err != nil {
		log.Println(err)
		return ""
	}
	return inviteResp.Callid
}

func SendBlk(blk []byte, is_first bool) int {
	base64Blk := base64.RawURLEncoding.EncodeToString(blk)
	streamUrl := StreamUrl + base64Blk
	if is_first {
		streamUrl += "&is_new=true"
	}
	resp, err := http.Get(streamUrl)
	if err != nil {
		log.Println(err)
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("not 200")
		return 0
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return 0
	}
	streamResp := &CreateStreamResp{}
	err = json.Unmarshal(body, streamResp)
	if err != nil {
		log.Println(err)
		return 0
	}
	return streamResp.Ssrc
}

func SendBye(callid string) {
	url := ByeUrl + callid
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("not 200")
		return
	}
}

func CalcBlkLen(len, pos int) int {
	blkLen := BlkLen
	left := len - pos
	if left < BlkLen {
		blkLen = left
	}
	return blkLen
}

func main() {
	log.SetFlags(log.Lshortfile)
	pcm, err := ioutil.ReadFile("/Users/rigensen/workspace/tmp/test.pcma")
	if err != nil {
		log.Println(err)
		return
	}
	pos := 0
	callid := ""
	is_first := true
	for pos < len(pcm) {
		blkLen := CalcBlkLen(len(pcm), pos)
		blk := pcm[pos : pos+blkLen]
		if is_first {
			ssrc := SendBlk(blk, true)
			log.Println("ssrc", ssrc)
			callid = InviteAudio(ssrc)
			is_first = false
		} else {
			SendBlk(blk, false)
		}
		pos += BlkLen
	}
	time.Sleep(time.Duration(1) * time.Second)
	SendBye(callid)
}

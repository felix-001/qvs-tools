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

const gbId = "31010403001370002272"
const chid = "31010403001370002272"
const RtpIp = "192.168.1.5"
const Host = "localhost"
const StreamPort = "1985"
const SipPort = "7279"
const BasePath = "api/v1/gb28181?"
const StreamBasePath = "http://" + Host + ":" + StreamPort + "/" + BasePath
const SipBasePath = "http://" + Host + ":" + SipPort + "/" + BasePath

const CreateChUrl = StreamBasePath + "action=create_audio_stream&id=" + gbId
const InviteUrl = SipBasePath + "action=sip_invite&chid=" + chid + "&id=" + gbId + "&ip=" + RtpIp + "&rtp_port=9001&rtp_proto=udp&is_talk=true&ssrc="
const ByeUrl = SipBasePath + "action=sip_bye&chid=" + chid + "&id=" + gbId + "&is_talk=true&call_id="
const DeleteChUrl = StreamBasePath + "action=delete_audio_stream&id=" + gbId
const QueryChUrl = StreamBasePath + "action=query_audio_channel&id=" + gbId
const AppendPcmUrl = StreamBasePath + "action=append_audio_pcm&id=" + gbId + "&base64_pcm="
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

func SendBlk(blk []byte) error {
	base64Blk := base64.RawURLEncoding.EncodeToString(blk)
	streamUrl := AppendPcmUrl + base64Blk
	resp, err := http.Get(streamUrl)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("not 200")
		return err
	}
	return nil
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

func DeleteAudioStream() error {
	resp, err := http.Get(DeleteChUrl)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("not 200")
		return err
	}
	return nil
}

type Channel struct {
	Id   string `json:"id"`
	Ssrc int    `json:"ssrc"`
}

type Data struct {
	Channels []Channel `json:"channels,omitempty"`
}

type QueryChResp struct {
	Code int  `json:"code"`
	Data Data `json:"data"`
}

func IsAudioChExist() bool {
	resp, err := http.Get(QueryChUrl)
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("not 200")
		return false
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false
	}
	queryResp := &QueryChResp{}
	err = json.Unmarshal(body, queryResp)
	if err != nil {
		log.Println(err)
		return false
	}
	if len(queryResp.Data.Channels) > 0 {
		return true
	}
	return false
}

func CreateAudioCh() (error, int) {
	resp, err := http.Get(CreateChUrl)
	if err != nil {
		log.Println(err)
		return err, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("not 200")
		return err, 0
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return err, 0
	}
	streamResp := &CreateStreamResp{}
	err = json.Unmarshal(body, streamResp)
	if err != nil {
		log.Println(err)
		return err, 0
	}
	return nil, streamResp.Ssrc
}

func main() {
	log.SetFlags(log.Lshortfile)
	pcm, err := ioutil.ReadFile("/Users/rigensen/workspace/tmp/test.pcma")
	if err != nil {
		log.Println(err)
		return
	}
	if IsAudioChExist() {
		err := DeleteAudioStream()
		if err != nil {
			return
		}
	}
	err, ssrc := CreateAudioCh()
	if err != nil {
		return
	}
	log.Println("ssrc", ssrc)
	callid := InviteAudio(ssrc)
	log.Println("callid:", callid)
	pos := 0
	for pos < len(pcm) {
		blkLen := CalcBlkLen(len(pcm), pos)
		blk := pcm[pos : pos+blkLen]
		err := SendBlk(blk)
		if err != nil {
			return
		}
		pos += BlkLen
	}
	time.Sleep(time.Duration(1) * time.Second)
	SendBye(callid)
}

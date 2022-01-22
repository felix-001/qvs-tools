package ts

import (
	"encoding/json"
	"fmt"
	"hlsdbg/utils"
	"io/ioutil"
	"log"
	"os/exec"
)

type TsMgr struct {
	index int
}

func New() *TsMgr {
	return &TsMgr{}
}

type Frame struct {
	MediaType      string `json:"media_type"`
	StreamIndex    int    `json:"stream_index"`
	KeyFrame       int    `json:"key_frame"`
	PktPts         int    `json:"pkt_pts"`
	PktPtsTime     string `json:"pkt_pts_time"`
	PktDts         int    `json:"pkt_dts"`
	PktDtsTime     string `json:"pkt_dts_time"`
	PktDuration    int    `json:"pkt_duration"`
	PktPos         string `json:"pkt_pos"`
	PktSize        int    `json:"pkt_sizeo"`
	SampleFmt      string `json:"sample_fmt"`
	NbSamples      int    `json:"nb_samples"`
	Channels       int    `json:"channels"`
	ChannelLaylout string `json:"channel_laylout"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	PixFmt         string `json:"pix_fmt"`
	PictType       string `json:"pict_type"`
}

type FrameInfo struct {
	Frames []Frame `json:"frames"`
}

func (self *TsMgr) parse(filename string) error {
	cmdstr := "ffprobe -loglevel quiet -show_frames -of json " + filename
	cmd := exec.Command("bash", "-c", cmdstr)
	jsonstr, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("cmd:", cmdstr, "err:", err)
		return err
	}
	frameInfo := FrameInfo{}
	err = json.Unmarshal(jsonstr, &frameInfo)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (self *TsMgr) Fetch(addr string) error {
	body, err := utils.HttpGet(addr)
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf("/tmp/%d.ts", self.index)
	err = ioutil.WriteFile(fileName, []byte(body), 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	if err := self.parse(fileName); err != nil {
		return err
	}
	self.index++
	return nil
}

package ts

import (
	"encoding/json"
	"errors"
	"fmt"
	"hlsdbg/utils"
	"io/ioutil"
	"log"
	"os/exec"
	"time"
)

var (
	ErrParseTS = errors.New("parse ts error")
)

type TsMgr struct {
	index              int
	wallClockStartTime int64
	ptsStart           int
	totalBytes         int
	totalFrames        int
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

func (self *TsMgr) parse(filename string) ([]Frame, error) {
	cmdstr := "ffprobe -loglevel quiet -show_frames -of json " + filename
	cmd := exec.Command("bash", "-c", cmdstr)
	jsonstr, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("cmd:", cmdstr, "err:", err)
		return nil, err
	}
	frameInfo := FrameInfo{}
	err = json.Unmarshal(jsonstr, &frameInfo)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return frameInfo.Frames, nil
}

func (self *TsMgr) Check(frames []Frame) {
	if self.wallClockStartTime == 0 {
		self.wallClockStartTime = time.Now().UnixMilli()
		self.ptsStart = frames[0].PktPts
	} else {
		len := len(frames)
		ptsDur := (frames[len-1].PktPts - self.ptsStart) / 90
		wallClockDur := time.Now().UnixMilli() - self.wallClockStartTime
		if wallClockDur > int64(ptsDur) {
			log.Println("playback stall, wallClockDur:", wallClockDur, "ms", "ptsDur:", ptsDur, "ms")
		}
	}

}

func (self *TsMgr) Fetch(addr string) ([]Frame, error) {
	body, cost, err := utils.HttpGet(addr)
	if err != nil {
		return nil, err
	}
	fileName := fmt.Sprintf("/tmp/%d.ts", self.index)
	err = ioutil.WriteFile(fileName, []byte(body), 0644)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	frames, err := self.parse(fileName)
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, ErrParseTS
	}
	tsDur := (frames[len(frames)-1].PktPts - frames[0].PktPts) / 90
	self.totalBytes += len(body) * 8 / 1024
	self.totalFrames += len(frames)
	wallClockDur := float64(time.Now().UnixMilli()-self.wallClockStartTime) / 1000
	bitrate := float64(self.totalBytes) / wallClockDur
	fps := float64(self.totalFrames) / wallClockDur
	log.Printf("cost: %dms ts size: %dk ts duration: %dms frame count: %d bitrate: %dKbps/s fps: %d total bytes: %dk\n",
		cost, len(body)/1024, tsDur, len(frames), int(bitrate), int(fps), self.totalBytes)
	self.index++
	return frames, nil
}

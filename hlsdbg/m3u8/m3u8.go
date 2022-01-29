package m3u8

import (
	"bytes"
	"errors"
	"hlsdbg/utils"
	"log"
	"strings"

	hls "github.com/grafov/m3u8"
)

var (
	errParseAM3u8      = errors.New("parse a.m3u8 error")
	errM3u8ListTypeErr = errors.New("m3u8 list type err")
)

type M3u8 struct {
	addr string
}

func New() *M3u8 {
	return &M3u8{}
}

func (self *M3u8) Init(addr string) error {
	body, err := utils.HttpGet(addr)
	if err != nil {
		return err
	}
	start := strings.Index(body, "http")
	if start == -1 {
		return errParseAM3u8
	}
	self.addr = body[start : len(body)-1]
	log.Println(self.addr)
	return nil
}

func (self *M3u8) Fetch() (*hls.MediaPlaylist, error) {
	body, cost, err := utils.HttpGet(self.addr)
	if err != nil {
		return nil, err
	}
	log.Println("cost:", cost)
	p, listType, err := hls.DecodeFrom(bytes.NewReader([]byte(body)), true)
	if err != nil {
		return nil, err
	}
	if listType != hls.MEDIA {
		return nil, errM3u8ListTypeErr
	}
	playlist := p.(*hls.MediaPlaylist)
	return playlist, nil
}

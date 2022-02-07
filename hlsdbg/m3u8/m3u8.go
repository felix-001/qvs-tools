package m3u8

import (
	"bytes"
	"errors"
	"fmt"
	"hlsdbg/utils"
	"net/url"
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
	body, _, err := utils.HttpGet(addr)
	if err != nil {
		return err
	}
	start := strings.Index(body, "http")
	if start == -1 {
		return errParseAM3u8
	}
	self.addr = body[start : len(body)-1]
	//log.Println(self.addr)
	return nil
}

func (self *M3u8) Fetch() (*hls.MediaPlaylist, string, error) {
	u, err := url.Parse(self.addr)
	if err != nil {
		return nil, "", err
	}
	host := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	body, _, err := utils.HttpGet(self.addr)
	if err != nil {
		return nil, "", err
	}
	//log.Println("cost:", cost, "ms")
	p, listType, err := hls.DecodeFrom(bytes.NewReader([]byte(body)), true)
	if err != nil {
		return nil, "", err
	}
	if listType != hls.MEDIA {
		return nil, "", errM3u8ListTypeErr
	}
	playlist := p.(*hls.MediaPlaylist)
	return playlist, host, nil
}

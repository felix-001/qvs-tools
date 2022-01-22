package m3u8

import (
	"errors"
	"hlsdbg/utils"
	"log"
	"strings"
)

var (
	errParserAM3u8 = errors.New("parse a.m3u8 error")
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
		return errParserAM3u8
	}
	self.addr = body[start : len(body)-1]
	log.Println(self.addr)
	return nil
}

func (self *M3u8) Fetch() ([]string, error) {
	return nil, nil
}

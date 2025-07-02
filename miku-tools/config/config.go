package config

import (
	"flag"
	"log"
	"os"

	qnconfig "github.com/qbox/bo-sdk/sdk/qconf/qconfapi/config"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
)

type Config struct {
	Cmd         string
	Uid         string
	Method      string
	Body        string
	Addr        string
	Help        bool
	Https       bool
	Pcdn        string
	User        string
	Passwd      string
	SchedIp     string
	Bucket      string
	Stream      string
	Domain      string
	Key         string
	Province    string
	SourceId    string
	OriginKey   string
	Origin      string
	Area        string
	Isp         string
	OriginKeyDy string
	ID          string
	OriginKeyHw string
	Format      string
	Node        string
	ConnId      string
	Ip          string
	Basesub     int
	SubStream   int
	Startid     int
	Port        int
	F           string
	T           string
	Ak          string          `json:"ak"`
	Sk          string          `json:"sk"`
	Secret      string          `json:"secret"`
	IPDB        ipdb.Config     `json:"ipdb"`
	RedisAddrs  []string        `json:"redis_addrs"`
	AccountCfg  qnconfig.Config `json:"acc"`
	DyApiSecret string          `json:"dy_api_secret"`
	DyApiDomain string          `json:"dy_api_domain"`
}

func Load() *Config {
	var conf Config
	_, err := os.Stat("/usr/local/etc/mikutool.json")
	if !os.IsNotExist(err) {
		if err = qconfig.LoadFile(&conf, "/usr/local/etc/mikutool.json"); err != nil {
			log.Fatalf("load config failed, err: %v", err)
		}
		return &conf
	}
	_, err = os.Stat("/tmp/mikutool.json")
	if os.IsNotExist(err) {
		log.Fatalf("load config failed, err: %v", err)
		return nil
	}
	if err = qconfig.LoadFile(&conf, "/tmp/mikutool.json"); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	return &conf
}

func (c *Config) ParseConsole() {
	flag.BoolVar(&c.Help, "h", false, "help")
	flag.StringVar(&c.Cmd, "cmd", "streams", "需要执行的命令")
	flag.StringVar(&c.Uid, "uid", "", "uid")
	flag.StringVar(&c.Method, "method", "", "method")
	flag.StringVar(&c.Body, "body", "", "body")
	flag.StringVar(&c.Addr, "addr", "", "addr")
	flag.StringVar(&c.Ak, "ak", "", "ak")
	flag.StringVar(&c.Sk, "sk", "", "sk")
	flag.StringVar(&c.Secret, "secret", "", "secret")
	flag.StringVar(&c.Pcdn, "pcdn", "", "pcdn")
	flag.StringVar(&c.Bucket, "bucket", "", "bucket")
	flag.StringVar(&c.Stream, "stream", "", "stream")
	flag.StringVar(&c.Domain, "domain", "", "domain")
	flag.StringVar(&c.SourceId, "source_id", "", "source_id")
	flag.StringVar(&c.OriginKey, "origin_key", "", "origin_key")
	flag.StringVar(&c.OriginKeyDy, "origin_key_dy", "", "origin_key_dy")
	flag.StringVar(&c.OriginKeyHw, "origin_key_hw", "", "origin_key_hw")
	flag.StringVar(&c.Origin, "origin", "", "origin")
	flag.StringVar(&c.Area, "area", "", "area")
	flag.StringVar(&c.Isp, "isp", "", "isp")
	flag.StringVar(&c.F, "f", "", "f")
	flag.StringVar(&c.T, "t", "", "t")
	flag.IntVar(&c.Basesub, "basesub", 0, "basesub")
	flag.IntVar(&c.SubStream, "substream", 0, "substream")
	flag.IntVar(&c.Startid, "startid", 0, "startid")

	flag.Parse()
}

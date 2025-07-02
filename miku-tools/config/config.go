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
	Pcdn        string
	Bucket      string
	Stream      string
	Domain      string
	SourceId    string
	OriginKey   string
	Origin      string
	Area        string
	Isp         string
	OriginKeyDy string
	OriginKeyHw string
	Basesub     int
	SubStream   int
	Startid     int
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
	flag.Parse()
}

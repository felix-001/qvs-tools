package config

import (
	"flag"
	"log"
	"os"

	qconfig "github.com/qiniu/x/config"
)

type Config struct {
	Cmd string
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
	flag.StringVar(&c.Cmd, "cmd", "streams", "需要执行的命令")
	flag.Parse()
}

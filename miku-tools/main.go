package main

import (
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := loadCfg()
	parser := newParser(conf)
	parser.init()
	parser.CmdMap[conf.Cmd].Handler()
}

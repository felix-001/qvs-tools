package main

import (
	"log"
	"middle-source-analysis/config"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := config.LoadCfg()
	parser := newParser(conf)
	parser.init()
	parser.CmdMap[conf.Cmd].Handler()
}

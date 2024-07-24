package main

import (
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := loadCfg()
	parser := newParser(conf)
	parser.init()

	if conf.Monitor {
		parser.nodeMonitor()
		return
	}

	if conf.Node != "" {
		parser.dumpNodeStreams()
		return
	}

	if conf.Stream != "" {
		parser.dumpStream()
		return
	}

	parser.dumpStreams()
}

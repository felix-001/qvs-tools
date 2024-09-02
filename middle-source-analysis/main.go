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

	if conf.Streams {
		parser.dumpStreams()
	}

	if conf.Bw {
		parser.CalcTotalBw()
	}

	if conf.LagFile != "" {
		parser.LagAnalysis()
	}

	if conf.Pcdn {
		parser.PcdnDbg()
	}

	if conf.DnsResFile != "" {
		parser.DnsChk()
	}

	if conf.PathqueryLogFile != "" {
		parser.pathqueryChk()
	}
}

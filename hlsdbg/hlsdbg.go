package main

import (
	"flag"
	"log"
	"os"

	"hlsdbg/m3u8"
	"hlsdbg/ts"
)

var addr *string

func parseConsole() {
	addr = flag.String("url", "", "hls播放地址")
	flag.Parse()
	if *addr == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	parseConsole()
	m3u8 := m3u8.New()
	tsMgr := ts.New()
	if err := m3u8.Init(*addr); err != nil {
		return
	}
	for {
		tslist, err := m3u8.Fetch()
		if err != nil {
			log.Println(err)
			return
		}
		for _, ts := range tslist {
			if err := tsMgr.Fetch(ts); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
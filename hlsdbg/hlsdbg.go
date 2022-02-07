package main

import (
	"flag"
	"fmt"
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
	var lastSeq uint64 = 0
	for {
		playlist, host, err := m3u8.Fetch()
		if err != nil {
			log.Println(err)
			return
		}
		if lastSeq == playlist.SeqNo {
			continue
		}
		log.Println("seqNo:", playlist.SeqNo)
		lastSeq = playlist.SeqNo
		for i := 0; i < int(playlist.Count()); i++ {
			addr := fmt.Sprintf("%s/%s", host, playlist.Segments[i].URI)
			frames, err := tsMgr.Fetch(addr)
			if err != nil {
				log.Println(err)
				return
			}
			tsMgr.Check(frames)
		}
	}
}

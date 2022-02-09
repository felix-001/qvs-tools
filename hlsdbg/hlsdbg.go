package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

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

// 检查每一个ts的所有帧时间戳是否正常
// 计算帧率
// 计算码率
// 两个sequence之间的时间间隔
func main() {
	log.SetFlags(log.Lshortfile)
	parseConsole()
	m3u8 := m3u8.New()
	tsMgr := ts.New()
	if err := m3u8.Init(*addr); err != nil {
		return
	}
	var lastSeq uint64 = 0
	var lastSeqTime int64 = 0
	for {
		playlist, host, err := m3u8.Fetch()
		if err != nil {
			log.Println(err)
			return
		}
		if lastSeq == playlist.SeqNo {
			continue
		}
		if lastSeqTime != 0 {
			dur := time.Now().UnixMilli() - lastSeqTime
			log.Println("seq gap:", dur, "ms")
		}
		lastSeqTime = time.Now().UnixMilli()
		log.Println("seqNo:", playlist.SeqNo)
		lastSeq = playlist.SeqNo
		for i := 0; i < int(playlist.Count()); i++ {
			addr := fmt.Sprintf("%s/%s", host, playlist.Segments[i].URI)
			frames, err := tsMgr.Fetch(addr)
			if err != nil && err != ts.ErrParseTS {
				log.Println(err)
				return
			}
			if err != ts.ErrParseTS {
				tsMgr.Check(frames)
			}
		}
	}
}

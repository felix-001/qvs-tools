package message

import (
	"fmt"
	"hlsdbg/ts"
	"log"

	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
)

var (
	index int
	win   *astilectron.Window
	msg   Message
)

const Max int = 100

func SetWindow(w *astilectron.Window) {
	win = w
}

type Message struct {
	Cost    [Max]int64
	TsDur   [Max]int
	TsSize  [Max]int
	Fps     [Max]float64
	Bitrate [Max]float64
	SeqGap  [Max]int64
}

func SendData(tsInfo *ts.TsInfo) {
	if index == Max {
		for i := 0; i < Max-1; i++ {
			msg.Cost[i] = msg.Cost[i+1]
			msg.TsDur[i] = msg.TsDur[i+1]
			msg.TsSize[i] = msg.TsSize[i+1]
			msg.Fps[i] = msg.Fps[i+1]
			msg.Bitrate[i] = msg.Bitrate[i+1]
			msg.SeqGap[i] = msg.SeqGap[i+1]
		}
	}
	msg.Cost[index] = tsInfo.Cost
	msg.TsDur[index] = tsInfo.TsDur
	msg.TsSize[index] = tsInfo.TsSize
	msg.Fps[index] = tsInfo.Fps
	msg.Bitrate[index] = tsInfo.Bitrate
	msg.SeqGap[index] = tsInfo.SeqGap
	if index < Max {
		index++
	}
	if err := bootstrap.SendMessage(win, "update", msg); err != nil {
		log.Println(fmt.Errorf("sending update event failed: %w", err))
	}
}

// handleMessages handles messages
func HandleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "disp":
		return nil, nil
	}
	return
}

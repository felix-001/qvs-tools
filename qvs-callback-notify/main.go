package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

type CallbackMsg struct {
	Type        string `json:"type"`
	Nsid        string `json:"nsId"`
	Gbid        string `json:"gbid"`
	ChId        string `json:"chGbid"`
	DeviceState string `json:"deviceState"`
	TimeSec     int    `json:"timeSec"`
	ReqId       string `json:"reqId"`
}

func runPy(param string) {
	cmd := exec.Command("python", "/root/liyq/sendmail.py", param)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("send mail fail", err, param, "out", string(b))
	} else {
		log.Println("send mail success", string(b))
	}
}

func callbackNotify(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("body:", string(body))
	var msg CallbackMsg
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Println(err)
		return
	}
	if msg.Type != "device" {
		return
	}
	mail := fmt.Sprintf("nsid: %s gbid: %s chid: %s state: %s time: %d reqId: %s", msg.Nsid,
		msg.Gbid, msg.ChId, msg.DeviceState, msg.TimeSec, msg.ReqId)
	log.Println(mail)
	//runPy(mail)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	http.HandleFunc("/callback/notify", callbackNotify)
	http.ListenAndServe("0.0.0.0:8082", nil)
}

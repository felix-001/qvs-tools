package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"time"
)

type Stream struct {
	Status bool `json:"status"`
}

func RunCmd(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return string(b), err
	}
	raw := string(b)
	//log.Println(raw)
	return raw, nil
}

var conf = "./alert.conf"

type AlertConf struct {
	Streams       []string `json:"streams"`
	SleepInterval int      `json:"interval"`
}

func loadConf() (alertConf AlertConf) {
	b, err := ioutil.ReadFile(conf)
	if err != nil {
		log.Fatalln(err)
	}
	if err := json.Unmarshal(b, &alertConf); err != nil {
		log.Fatalln(err)
	}
	return
}

func getStreamStatus(streamId string) (bool, error) {
	cmd := fmt.Sprintf("curl -s --location --request GET http://10.20.76.42:7277/v1/streams/%s --header 'authorization: QiniuStub uid=1'", streamId)
	result, err := RunCmd(cmd)
	if err != nil {
		log.Println(result, err, cmd)
		return false, err
	}
	stream := &Stream{}
	if err := json.Unmarshal([]byte(result), stream); err != nil {
		return false, err
	}
	return stream.Status, err
}

var wecomNotifyUrl = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=92f4334a-906a-4394-b975-d9bba071f19d"

type WeComNotfiy struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conf := loadConf()
	cnt := 0
	for {
		for _, streamId := range conf.Streams {
			status, err := getStreamStatus(streamId)
			if err != nil {
				log.Println("get stream status err", streamId, err)
				continue
			}
			if !status {
				content := fmt.Sprintf("流id: %s已离线", streamId)
				notify := WeComNotfiy{
					Msgtype: "text",
					Text: struct {
						Content string `json:"content"`
					}{Content: content},
				}
				body, err := json.Marshal(&notify)
				if err != nil {
					log.Println(err)
				}
				resp, err := http.Post(wecomNotifyUrl, "application/json", bytes.NewReader(body))
				if err != nil {
					log.Println(err)
				}
				if resp.StatusCode != 200 {
					log.Println("http code err:", resp.StatusCode, resp.Status)
				}
			}
		}
		time.Sleep(time.Duration(conf.SleepInterval) * time.Second)
		cnt++
		log.Println("running", cnt, "times")
	}
}

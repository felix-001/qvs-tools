package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
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
	if strings.Contains(raw, "Pseudo-terminal") {
		new := ""
		ss := strings.Split(raw, "\n")
		if len(ss) == 1 {
			return "", nil
		}
		for _, str := range ss {
			if strings.Contains(str, "Pseudo-terminal") {
				continue
			}
			if len(str) == 0 {
				continue
			}
			//log.Println("str len:", len(str))
			new += str + "\r\n"
		}
		//log.Println("new:", new)
		return new, nil
	}
	//log.Println(raw)
	return raw, nil
}

func message(content string) {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell app "System Events" to display dialog "%s" buttons {"OK"} default button "OK"`, content))
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func RunJumpboxCmd(rawCmd string) (string, error) {
	jumpbox := "ssh -t liyuanquan@10.20.34.27"
	cmd := fmt.Sprintf("%s \" %s \"", jumpbox, rawCmd)
	return RunCmd(cmd)
}

var conf = "./alert.conf"

type AlertConf struct {
	Streams       []string `json:"streams"`
	SleepInterval int      `json:"interval"`
	NotifyMethod  string   `json:"notify_method"`
}

var globalConf AlertConf

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
	//cmd := fmt.Sprintf("curl -s --location --request GET http://10.20.76.42:7277/v1/streams/%s --header 'authorization: QiniuStub uid=1'", streamId)
	cmd := fmt.Sprintf("curl -s --location --request GET http://10.20.76.42:7277/v1/streams/%s --header 'authorization: QiniuStub uid=1'", streamId)
	result, err := RunJumpboxCmd(cmd)
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

var wecomNotifyUrl = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=59b3d537-8160-4870-ba92-31245c4b4729"

type WeComNotfiy struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func wecomNotify(content string) {
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	globalConf = loadConf()
	cnt := 0
	for {
		for _, streamId := range globalConf.Streams {
			status, err := getStreamStatus(streamId)
			if err != nil {
				log.Println("get stream status err", streamId, err)
				continue
			}
			if !status {
				content := fmt.Sprintf("%v 流id: %s已离线", time.Now(), streamId)
				if globalConf.NotifyMethod == "wecom" {
					wecomNotify(content)
				} else {
					message(content)
				}

			}
		}
		time.Sleep(time.Duration(globalConf.SleepInterval) * time.Second)
		cnt++
		log.Println("running", cnt, "times")
	}
}

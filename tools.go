package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func Exec(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(cmdstr, err)
		return "", err
	}
	return string(b), err
}

func (self *LogManager) GetLatestLogFile() (string, error) {
	cmdstr := "ls -t " + self.logPath + "/qvs-sip.log* | head -n 1"
	logFile, err := Exec(cmdstr)
	if err != nil {
		return "", err
	}
	return logFile[:len(logFile)-1], err
}

type LogParser struct {
	logFile string
	gbid    string
	chid    string
	logPath string
}

func NewLogParser(logFile, gbid, chid, logPath string) *LogParser {
	return &LogParser{
		logFile: logFile,
		gbid:    gbid,
		chid:    chid,
		logPath: logPath}
}

type InviteInfo struct {
	ssrc  string
	rtpIp string
	time  string
}

func (self *LogParser) getLastInviteLog() (string, error) {
	cmdstr := "tac " + self.logFile + " | grep \"sip_invite&chid=" + self.chid + "&id=" + self.gbid + "\" -m 1"
	return Exec(cmdstr)
}

func (self *LogManager) GetLogsFromJJh1445() (string, error) {
	cmdstr := "qscp qboxserver@jjh1445:/home/qboxserver/qvs-sip/_package/run/qvs-sip.log* " + self.logPath
	return Exec(cmdstr)
}

func (self *LogParser) GetInviteInfo() (info *InviteInfo, err error) {
	inviteLog, err := self.getLastInviteLog()
	if err != nil {
		return nil, err
	}
	start := strings.Index(inviteLog, "[2021")
	if start == -1 {
		log.Println("get start error")
		return nil, errors.New("get start error")
	}
	end := strings.Index(inviteLog, "]")
	if end == -1 {
		log.Println("get end error")
		return nil, errors.New("get end error")
	}
	info = &InviteInfo{}
	info.time = inviteLog[start+1 : end]

	start = strings.Index(inviteLog, "ip=")
	end = strings.Index(inviteLog, "&rtp_por")
	info.rtpIp = inviteLog[start+3 : end]

	start = strings.Index(inviteLog, "ssrc=")
	end = strings.Index(inviteLog, "&token")
	info.ssrc = inviteLog[start+5 : end]

	return
}

type LogManager struct {
	logPath string
}

func NewLogManager(logPath string) *LogManager {
	return &LogManager{logPath: logPath}
}

func getNodeIdFromPdr() {

}

// todo
// pdr搜索
// [2021-05-31 23:20:18.672][Trace][24719][733g0873] HTTP #0 127.0.0.1:12660 GET http://42.123.110.67:1985/api/v1/gb28181?action=create_channel&app=3nm4x0ulb1fl0&g7112aac_enable=go&id=31011500991180001563_34020000001310000111&port_mode=fixed&protocol=udp&push=42.123.110.67&real=222.208.242.45&token=CZsxhjEqdmcNMrSQfUqM, content-length=-1
// 这个日志，去获取rtp node id

func main() {
	log.SetFlags(log.Lshortfile)
	logPath := flag.String("logpath", "~/logs", "log file path")
	gbid := flag.String("gbid", "", "gbid")
	chid := flag.String("chid", "", "chid")
	flag.Parse()
	logMgr := NewLogManager(*logPath)
	res, err := logMgr.GetLogsFromJJh1445()
	fmt.Println(res)
	if err != nil {
		log.Println(err)
		return
	}
	logFile, _ := logMgr.GetLatestLogFile()
	log.Println(logFile)
	parser := NewLogParser(logFile, *gbid, *chid, *logPath)
	inviteInfo, err := parser.GetInviteInfo()
	if err != nil {
		return
	}
	log.Printf("%+v\n", inviteInfo)
}

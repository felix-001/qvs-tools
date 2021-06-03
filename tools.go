package main

import (
	"errors"
	"flag"
	"log"
	"os/exec"
	"strings"
)

func Exec(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("cmd:", cmdstr, "err:", err)
		return "", err
	}
	return string(b), err
}

func (self *LogManager) GetLatestSipLogFile() (string, error) {
	cmdstr := "ls -t " + self.logPath + "/qvs-sip.log* | head -n 1"
	logFile, err := Exec(cmdstr)
	if err != nil {
		return "", err
	}
	return logFile[:len(logFile)-1], err
}

func (self *LogManager) GetRtpLogFromNode(rtpNode string) error {
	cmdstr := "qscp qboxserver@" + rtpNode + ":/home/qboxserver/qvs-rtp/_package/run/qvs-rtp.log* ~/logs/"
	_, err := Exec(cmdstr)
	return err
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
	s, err := Exec(cmdstr)
	return s, err
}

func (self *LogManager) GetLogsFromJJh1445() (string, error) {
	cmdstr := "qscp qboxserver@jjh1445:/home/qboxserver/qvs-sip/_package/run/qvs-sip.log* " + self.logPath
	return Exec(cmdstr)
}

func (self *LogParser) GetInviteInfo() (info *InviteInfo, err error) {
	inviteLog, _err := self.getLastInviteLog()
	if _err != nil {
		err = _err
		return
	}
	start := strings.Index(inviteLog, "[2021")
	if start == -1 {
		log.Println("get start error")
		err = errors.New("get start error")
		return
	}
	end := strings.Index(inviteLog, "]")
	if end == -1 {
		log.Println("get end error")
		err = errors.New("get end error")
		return
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

func (self *LogParser) GetNodeIdFromPdr(rtpIp string) (string, error) {
	cmdstr := "~/bin/getRtpNodeId.py " + rtpIp
	res, err := Exec(cmdstr)
	if err != nil {
		return "", err
	}
	return res[:len(res)-1], nil
}

// 拉流失败
// 1.有没有503
// 2.有没有inviting
// 3.tcp/udp有没有收到包
// 4.connect reset by peer?
// 5.ssrc illegal?

func main() {
	log.SetFlags(log.Lshortfile)
	logPath := flag.String("logpath", "~/logs", "log file path")
	gbid := flag.String("gbid", "", "gbid")
	chid := flag.String("chid", "", "chid")
	flag.Parse()
	if *gbid == "" || *chid == "" {
		log.Println("must pass gbid and chid")
		return
	}
	logMgr := NewLogManager(*logPath)
	log.Println("start to fetch log file from jjh1445 ~/qvs-sip/_package/run")
	_, err := logMgr.GetLogsFromJJh1445()
	//fmt.Println(res)
	if err != nil {
		log.Println(err)
		return
	}
	logFile, err := logMgr.GetLatestSipLogFile()
	if err != nil {
		return
	}
	log.Println("fetch log file: " + logFile + " done")
	parser := NewLogParser(logFile, *gbid, *chid, *logPath)
	inviteInfo, err := parser.GetInviteInfo()
	if err != nil {
		log.Println("get invite log err")
		return
	}
	log.Printf("%+v\n", inviteInfo)
	nodeId, err := parser.GetNodeIdFromPdr(inviteInfo.rtpIp)
	if err != nil {
		return
	}
	log.Println("rtp NodeId:", nodeId)
	log.Println("start to get qvs-rtp log from", nodeId)
	err = logMgr.GetRtpLogFromNode(nodeId)
	if err != nil {
		return
	}
	log.Println("fetch qvs-rtp log done")
}

package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"os/exec"
	"strings"
	"time"
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

func (self *LogManager) GetLatestRtpLogFile() (string, error) {
	cmdstr := "ls -t " + self.logPath + "/qvs-rtp.log* | head -n 1"
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
	inviteTime string
	logFile    string
	gbid       string
	chid       string
	logPath    string
	logMgr     *LogManager
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

func (self *LogParser) GetCreateChannelLogs(rtpLogFile string) (string, error) {
	cmdstr := "tac " + rtpLogFile +
		" | grep \"action=create_channel.*id=" +
		self.gbid + "_" + self.chid + "\" -m 10 "
	return Exec(cmdstr)
}

func (self *LogParser) GetTimeFromLog(line string) (string, error) {
	start := strings.Index(line, "[2021")
	if start == -1 {
		log.Println("get start error")
		return "", errors.New("get start error")
	}
	end := strings.Index(line, "]")
	if end == -1 {
		log.Println("get end error")
		return "", errors.New("get end error")
	}
	time := line[start+1 : end]
	return time, nil
}

var timeLayoutStr = "2006-01-02 15:04:05"

func TimeStr2ts(timeStr string) (int64, error) {
	loc, _ := time.LoadLocation("Local")
	res, err := time.ParseInLocation(timeLayoutStr, timeStr, loc)
	if err != nil {
		return 0, err
	}
	return res.UnixNano() / 1e6, nil
}

func (self *LogParser) getLogs(logFile, pattern string) (string, error) {
	cmdstr := "tac " + logFile +
		" | grep " + pattern +
		"\" -m 10"
	return Exec(cmdstr)
}

// 从指定行向下搜索日志
// tail -n 100 qvs-sip.log-0528101425 | grep qvs-sip | head -n 1
func (self *LogParser) SearchLog(logFile, pattern string, lineNo int) (string, error) {

}

// 搜索10行是否需要可配置
func (self *LogParser) GetCreateChannelTime(rtpLogFile string) (string, error) {
	logs, err := self.GetCreateChannelLogs(rtpLogFile)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(logs))
	inviteTimeStamp, err := TimeStr2ts(self.inviteTime)
	if err != nil {
		return "", err
	}
	for scanner.Scan() {
		time, err := self.GetTimeFromLog(scanner.Text())
		if err != nil {
			return "", err
		}
		ts, err := TimeStr2ts(time)
		if err != nil {
			return "", err
		}
		if ts > inviteTimeStamp {
			log.Println("skip", ts, time)
			continue
		}
		if inviteTimeStamp-ts < 1000 {
			return time, nil
		}
	}
	log.Println("create_channel not found")
	return "", errors.New("create_channel not found")
}

// 拉流失败
// 1.有没有503
// 2.有没有inviting
// 3.tcp/udp有没有收到包
// 4.connect reset by peer?
// 5.ssrc illegal?
// 6.tcp attach
// 7.是否有第二个create_channel
// 8.是否有delete channel

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
	parser.inviteTime = inviteInfo.time
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
	rtpLogFile, err := logMgr.GetLatestRtpLogFile()
	if err != nil {
		return
	}
	log.Println("rtp log file:", rtpLogFile)
	createChannelTime, err := parser.GetCreateChannelTime(rtpLogFile)
	if err != nil {
		return
	}
	log.Println("create channel time:", createChannelTime)

}

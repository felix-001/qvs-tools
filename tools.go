package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"os/exec"
	"strconv"
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
	cmdstr := "~/bin/GetLastLogFile.sh qvs-sip"
	logFile, err := Exec(cmdstr)
	if err != nil {
		return "", err
	}
	logFile = "~/logs/" + logFile[:len(logFile)-1]
	return logFile, err
}

func (self *LogManager) GetLatestRtpLogFile() (string, error) {
	cmdstr := "~/bin/GetLastLogFile.sh qvs-rtp"
	logFile, err := Exec(cmdstr)
	if err != nil {
		return "", err
	}
	logFile = "~/logs/" + logFile[:len(logFile)-1]
	return logFile, err
}

func (self *LogManager) GetRtpLogFromNode(rtpNode string) error {
	cmdstr := "qscp qboxserver@" + rtpNode + ":/home/qboxserver/qvs-rtp/_package/run/qvs-rtp.log* ~/logs/"
	_, err := Exec(cmdstr)
	return err
}

type LogParser struct {
	logMgr              *LogManager
	inviteTime          string
	logFile             string
	gbid                string
	chid                string
	logPath             string
	rtpLogFile          string
	createChannelLineNo int
	ssrc                string
	streamId            string
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
		" | grep -n \"action=create_channel.*id=" +
		self.gbid + "_" + self.chid + "\" -m 10 "
	return Exec(cmdstr)
}

func (self *LogParser) GetTimeLineFromLog(line, rtpLogFile, direction string) (int, string, error) {
	log.Println("line:", line)
	end := strings.Index(line, ":")
	if end == -1 {
		return 0, "", errors.New("get : error")
	}
	lineNoStr := line[:end]
	start := strings.Index(line, "[2021")
	if start == -1 {
		log.Println("get start error")
		return 0, "", errors.New("get start error")
	}
	end = strings.Index(line, "]")
	if end == -1 {
		log.Println("get end error")
		return 0, "", errors.New("get end error")
	}
	time := line[start+1 : end]
	maxLine, err := self.GetMaxLineNumOfFile(rtpLogFile)
	if err != nil {
		return 0, "", err
	}
	lineNo, err := strconv.Atoi(lineNoStr)
	if err != nil {
		return 0, "", errors.New("str to int error")
	}
	//log.Println("maxLine:", maxLine)
	if direction == "up" { // up/down
		lineNo = maxLine - lineNo + 1
	} else {
		lineNo = lineNo - 1
	}
	return lineNo, time, nil
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
func (self *LogParser) SearchLog(logFile, pattern string, startLineNo int) (string, error) {
	cmdstr := "tail -n " + strconv.Itoa(startLineNo) + " " +
		logFile + " | grep -n \"" + pattern + "\" | head -n 1"
	//log.Println(cmdstr)
	return Exec(cmdstr)
}

func (self *LogParser) SearchRtpLog(pattern string) (int, string, error) {
	_log, err := self.SearchLog(self.rtpLogFile, pattern, self.createChannelLineNo)
	if len(_log) == 0 {
		log.Println("search", pattern, "not found")
		return 0, "", errors.New("not found")
	}
	lineNo, time, err := self.GetTimeLineFromLog(_log, self.rtpLogFile, "down")
	if err != nil {
		return 0, "", err
	}
	return lineNo, time, nil
}

func (self *LogParser) SearchTcpAttachLog() (int, string, error) {
	pattern := "gb28181: tcp attach new stream channel id:" + self.streamId +
		" ssrs: " + self.ssrc
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchUdpPktLog() (int, string, error) {
	pattern := "gb28181 rtp enqueue : client_id " + self.streamId
	return self.SearchRtpLog(pattern)
}

// 搜索10行是否需要可配置
func (self *LogParser) GetCreateChannelTimeLine(rtpLogFile string) (int, string, error) {
	logs, err := self.GetCreateChannelLogs(rtpLogFile)
	//log.Println(logs)
	if err != nil {
		return 0, "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(logs))
	inviteTimeStamp, err := TimeStr2ts(self.inviteTime)
	if err != nil {
		return 0, "", err
	}
	for scanner.Scan() {
		line, time, err := self.GetTimeLineFromLog(scanner.Text(), rtpLogFile, "up")
		if err != nil {
			return 0, "", err
		}
		ts, err := TimeStr2ts(time)
		if err != nil {
			return 0, "", err
		}
		if ts > inviteTimeStamp {
			log.Println("skip", ts, time)
			continue
		}
		if inviteTimeStamp-ts < 1000 {
			return line, time, nil
		}
	}
	log.Println("create_channel not found")
	return 0, "", errors.New("create_channel not found")
}

func (self *LogParser) GetMaxLineNumOfFile(file string) (int, error) {
	cmdstr := "cat " + file + " | wc -l"
	res, err := Exec(cmdstr)
	if err != nil {
		return 0, err
	}
	ret, err := strconv.Atoi(res[:len(res)-1])
	if err != nil {
		log.Println("str to int err:", err)
		return 0, err
	}
	return ret, nil
}

func (self *LogParser) SaveStreamId(gbid, chid string) {
	if chid == "" {
		self.streamId = gbid
	} else {
		self.streamId = gbid + "_" + chid
	}
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
	parser.SaveStreamId(*gbid, *chid)
	inviteInfo, err := parser.GetInviteInfo()
	if err != nil {
		log.Println("get invite log err")
		return
	}
	log.Printf("%+v\n", inviteInfo)
	parser.inviteTime = inviteInfo.time
	parser.ssrc = inviteInfo.ssrc
	nodeId, err := parser.GetNodeIdFromPdr(inviteInfo.rtpIp)
	if err != nil {
		return
	}
	if nodeId == "Not found" {
		log.Println("rtp ip:", inviteInfo.rtpIp, "not found nodeId")
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
	parser.rtpLogFile = rtpLogFile
	lineNo, createChannelTime, err := parser.GetCreateChannelTimeLine(rtpLogFile)
	if err != nil {
		return
	}
	parser.createChannelLineNo = lineNo
	log.Println("create channel time:", createChannelTime, "lineNo:", lineNo)
	lineNo, time, err := parser.SearchTcpAttachLog()
	if err == nil {
		log.Println("tcp attach line no:", lineNo, "time:", time)
	} else {
		log.Println("tcp attach not found")
	}
	lineNo, time, err = parser.SearchUdpPktLog()
	if err == nil {
		log.Println("got udp pkt line no:", lineNo, "time:", time)
	} else {
		log.Println("not got udp pkt")
	}
}

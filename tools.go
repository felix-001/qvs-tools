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
	sipLogFile          string
	gbid                string
	chid                string
	logPath             string
	rtpLogFile          string
	createChannelLineNo int
	ssrc                string
	streamId            string
	sipSessionId        string
	sipInviteLineNo     int
	createChTime        string
}

func NewLogParser(logFile, gbid, chid, logPath string) *LogParser {
	return &LogParser{
		sipLogFile: logFile,
		gbid:       gbid,
		chid:       chid,
		logPath:    logPath}
}

type InviteInfo struct {
	ssrc  string
	rtpIp string
	time  string
}

func (self *LogParser) getLastInviteLog() (string, error) {
	cmdstr := "tac " + self.sipLogFile + " | grep -n \"sip_invite&chid=" + self.chid + "&id=" + self.gbid + "\" -m 1"
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
	logInfo, err := self.ParseLog(inviteLog, self.sipLogFile, "up")
	if err != nil {
		return nil, err
	}
	log.Println("sip session id:", logInfo.sessionId)
	log.Println("sip invite line num:", logInfo.lineNo)
	self.sipSessionId = logInfo.sessionId
	self.sipInviteLineNo = logInfo.lineNo
	info = &InviteInfo{}
	info.time = logInfo.time

	start := strings.Index(inviteLog, "ip=")
	end := strings.Index(inviteLog, "&rtp_por")
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

type LogInfo struct {
	lineNo    int
	time      string
	sessionId string
	duration  int64
}

func (self *LogParser) GetLineNoFromLog(line, logFile, direction string) (int, error) {
	end := strings.Index(line, ":")
	if end == -1 {
		return 0, errors.New("get : error")
	}
	lineNoStr := line[:end]
	maxLine, err := self.GetMaxLineNumOfFile(logFile)
	if err != nil {
		return 0, err
	}
	lineNo, err := strconv.Atoi(lineNoStr)
	if err != nil {
		err = errors.New("str to int error")
	}
	//log.Println("maxLine:", maxLine)
	if direction == "up" { // up/down
		lineNo = maxLine - lineNo + 1
	} else {
		lineNo = lineNo - 1
	}
	return lineNo, nil
}

func (self *LogParser) GetTimeFromLog(line, rtpLogFile, direction string) (string, error) {
	start := strings.Index(line, "[2021")
	if start == -1 {
		log.Println("get start error, line:", line)
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

func (self *LogParser) GetSessionIdFromLog(line, rtpLogFile, direction string) (string, error) {
	// 过滤掉非法字符
	//log.Println("line:", line)
	pos := strings.Index(line, "[2021")
	start := pos + 1
	for i := 0; i < 3; i++ {
		pos = strings.Index(line[start:], "[")
		if pos == -1 {
			log.Println("get start error, line:", line)
			return "", errors.New("get start error")
		}
		start += 1 + pos
		//log.Println("start:", start)
	}
	//start += 1 + pos
	end := strings.Index(line[start:], "]")
	if end == -1 {
		log.Println("get end error, line:", line, "start:", start)
		return "", errors.New("get end error")
	}
	sessionId := line[start : start+end]
	return sessionId, nil
}

func (self *LogParser) ParseLog(line, logFile, direction string) (logInfo *LogInfo, err error) {
	//log.Println("line:", line)
	logInfo = &LogInfo{}
	lineNo, err := self.GetLineNoFromLog(line, logFile, direction)
	if err != nil {
		return
	}
	logInfo.lineNo = lineNo

	time, err := self.GetTimeFromLog(line, logFile, direction)
	if err != nil {
		return
	}
	logInfo.time = time

	sessionId, err := self.GetSessionIdFromLog(line, logFile, direction)
	if err != nil {
		return
	}
	logInfo.sessionId = sessionId
	return
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

func GetDuration(time1 string, time2 string) int64 {
	t1, _ := TimeStr2ts(time1)
	t2, _ := TimeStr2ts(time2)
	return (t2 - t1)
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

func (self *LogParser) SearchRtpLog(pattern string) (*LogInfo, error) {
	_log, err := self.SearchLog(self.rtpLogFile, pattern, self.createChannelLineNo)
	if len(_log) == 0 {
		//log.Println("search", pattern, "not found")
		return nil, errors.New("not found")
	}
	logInfo, err := self.ParseLog(_log, self.rtpLogFile, "down")
	logInfo.lineNo += self.createChannelLineNo
	if err != nil {
		return nil, err
	}
	logInfo.duration = GetDuration(self.createChTime, logInfo.time)
	return logInfo, nil
}

func (self *LogParser) SearchSipLog(pattern string) (*LogInfo, error) {
	_log, err := self.SearchLog(self.sipLogFile, pattern, self.sipInviteLineNo)
	if len(_log) == 0 {
		//log.Println("search", pattern, "not found")
		return nil, errors.New("not found")
	}
	logInfo, err := self.ParseLog(_log, self.sipLogFile, "down")
	if err != nil {
		return nil, err
	}
	logInfo.lineNo += self.sipInviteLineNo
	logInfo.duration = GetDuration(self.inviteTime, logInfo.time)
	return logInfo, nil
}

func (self *LogParser) SearchInviteRespLog() (*LogInfo, error) {
	pattern := "INVITE response " + self.chid + "client status="
	return self.SearchSipLog(pattern)
}

func (self *LogParser) SearchInviteErrStateLog() (*LogInfo, error) {
	pattern := "error device->invite sipid =" + self.gbid + " " + self.chid + " state:"
	return self.SearchSipLog(pattern)
}

func (self *LogParser) SearchInviteDeviceOfflineLog() (*LogInfo, error) {
	pattern := "device " + self.chid + " offline"
	return self.SearchSipLog(pattern)
}

func (self *LogParser) SearchTcpAttachLog() (*LogInfo, error) {
	pattern := "gb28181: tcp attach new stream channel id:" + self.streamId +
		" ssrs:" + self.ssrc
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchUdpPktLog() (*LogInfo, error) {
	pattern := "gb28181 rtp enqueue : client_id " + self.streamId
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchSsrcIllegalLog() (*LogInfo, error) {
	pattern := "ssrc illegal on tcp payload chaanellid:" + self.streamId
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchConnectionResetByPeerLog() (*LogInfo, error) {
	pattern := "read() [src/protocol/srs_service_st.cpp:524][errno=104] chid: " + self.streamId
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchDeleteChannelLog() (*LogInfo, error) {
	pattern := "action=delete_channel&id=" + self.streamId
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchStreamH265Log() (*LogInfo, error) {
	pattern := "gb28181 gbId " + self.streamId + ", ps map video es_type=h265"
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchLostPktLog() (*LogInfo, error) {
	pattern := "gb28181: client_id " + self.streamId + " decode ps packet"
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchRtmpConnect() (*LogInfo, error) {
	pattern := "rtmp connect ok url=rtmp.*" + self.streamId
	return self.SearchRtpLog(pattern)
}

func (self *LogParser) SearchDecodePs() (*LogInfo, error) {
	pattern := "gb28181 gbId " + self.streamId + ", ps map video es_type="
	return self.SearchRtpLog(pattern)
}

func (self *LogManager) DeleteOldLogs() {
	cmdstr := "rm -rf ~/logs/*"
	Exec(cmdstr)
}

// 搜索10行是否需要可配置
func (self *LogParser) GetCreateChannelInfo(rtpLogFile string) (*LogInfo, error) {
	logs, err := self.GetCreateChannelLogs(rtpLogFile)
	//log.Println(logs)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(logs))
	inviteTimeStamp, err := TimeStr2ts(self.inviteTime)
	if err != nil {
		return nil, err
	}
	for scanner.Scan() {
		logInfo, err := self.ParseLog(scanner.Text(), rtpLogFile, "up")
		if err != nil {
			return nil, err
		}
		ts, err := TimeStr2ts(logInfo.time)
		if err != nil {
			return nil, err
		}
		if ts > inviteTimeStamp {
			log.Println("skip", ts, logInfo.time)
			continue
		}
		if inviteTimeStamp-ts < 1000 {
			return logInfo, nil
		}
	}
	log.Println("create_channel not found")
	return nil, errors.New("create_channel not found")
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

func (self *LogParser) GetLogs() {
	logInfo, err := self.SearchTcpAttachLog()
	if err == nil {
		log.Println("tcp attach line no:", logInfo.lineNo, "time:", logInfo.time)
	} else {
		log.Println("没有收到tcp连接")
	}
	logInfo, err = self.SearchSsrcIllegalLog()
	if err == nil {
		log.Println("got ssrc illegal line no:", logInfo.lineNo, "time:", logInfo.time)
	} else {
		//log.Println("not got ssrc illegal log")
	}
	logInfo, err = self.SearchConnectionResetByPeerLog()
	if err == nil {
		log.Println("got connection reset by peer line no:", logInfo.lineNo, "time:", logInfo.time)
	} else {
		//log.Println("not got connection by peer log")
	}
	logInfo, err = self.SearchDeleteChannelLog()
	if err == nil {
		log.Println("got delete channel line no:", logInfo.lineNo, "time:", logInfo.time)
	} else {
		//log.Println("not got delete channel log")
	}
	logInfo, err = self.SearchStreamH265Log()
	if err == nil {
		log.Println("got stream h265 line no:", logInfo.lineNo, "time:", logInfo.time)
	} else {
		//log.Println("not got stream h265 log")
	}
	logInfo, err = self.SearchLostPktLog()
	if err == nil {
		log.Println("got lost pkt line no:", logInfo.lineNo, "time:", logInfo.time)
	} else {
		//log.Println("not got lost pkt log")
	}
	logInfo, err = self.SearchInviteRespLog()
	if err == nil {
		log.Println("got invite resp log line no:", logInfo.lineNo, "time:", logInfo.time)
	}
	logInfo, err = self.SearchInviteErrStateLog()
	if err == nil {
		log.Println("got invite err state line no:", logInfo.lineNo, "time:", logInfo.time)
	}
	logInfo, err = self.SearchInviteDeviceOfflineLog()
	if err == nil && logInfo.sessionId == self.sipSessionId {
		log.Println("after", logInfo.duration, "ms got invite device offline line num:", logInfo.lineNo, "time:", logInfo.time)
	}
	logInfo, err = self.SearchRtmpConnect()
	if err == nil {
		log.Println("after", logInfo.duration, "ms got rtmp connect line num:", logInfo.lineNo, "time:", logInfo.time)
	}
	logInfo, err = self.SearchDecodePs()
	if err == nil {
		log.Println("after", logInfo.duration, "ms got decode ps line number:", logInfo.lineNo, "time:", logInfo.time)
		log.Println("收到rtp over udp 包")
		return
	}
	logInfo, err = self.SearchUdpPktLog()
	if err == nil {
		log.Println("got udp pkt line no:", logInfo.lineNo, "time:", logInfo.time)
	} else {
		log.Println("没有收到rtp over udp包")
	}
}

// todo
// 拉流失败
// 1.有没有503 --- ok
// 2.有没有inviting, err state=3 --- ok
// 3.tcp/udp有没有收到包 --- ok
// 4.connect reset by peer? --- ok
// 5.ssrc illegal? --- ok
// 6.tcp attach --- ok
// 7.是否有第二个create_channel
// 8.是否有delete channel  --- ok
// 9.h265? --- ok
// 10. 丢包？ --- ok
// 11. device offline --- ok
// 12. 丢包率
// 13. 拉流慢， rtmp connect --- ok
// 14. tcp gb281 create channel fail channelid:31011500991180000953_34020000001320000007 has exists(Resource temporarily unavailable)

func (self *LogManager) fetchSipLogs() error {
	self.DeleteOldLogs()
	log.Println("start to fetch log file from jjh1445 ~/qvs-sip/_package/run")
	_, err := self.GetLogsFromJJh1445()
	//fmt.Println(res)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)
	logPath := flag.String("logpath", "~/logs", "log file path")
	gbid := flag.String("gbid", "", "gbid")
	chid := flag.String("chid", "", "chid")
	reFetchLog := flag.Bool("refetch", false, "refetch log")
	flag.Parse()
	if *gbid == "" || *chid == "" {
		log.Println("must pass gbid and chid")
		return
	}
	logMgr := NewLogManager(*logPath)
	if *reFetchLog {
		err := logMgr.fetchSipLogs()
		if err != nil {
			return
		}
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
	log.Println("ssrc:", inviteInfo.ssrc)
	log.Println("rtp ip:", inviteInfo.rtpIp)
	log.Println("invite time:", inviteInfo.time)
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
	if *reFetchLog {
		log.Println("start to fetch qvs-rtp log from", nodeId)
		err = logMgr.GetRtpLogFromNode(nodeId)
		if err != nil {
			return
		}
		log.Println("fetch qvs-rtp log done")
	}
	rtpLogFile, err := logMgr.GetLatestRtpLogFile()
	if err != nil {
		return
	}
	log.Println("rtp log file:", rtpLogFile)
	parser.rtpLogFile = rtpLogFile
	logInfo, err := parser.GetCreateChannelInfo(rtpLogFile)
	if err != nil {
		return
	}
	parser.createChannelLineNo = logInfo.lineNo
	parser.createChTime = logInfo.time
	log.Println("create channel time:", logInfo.time, "lineNo:", logInfo.lineNo)
	parser.GetLogs()
}

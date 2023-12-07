package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type InvitInfo struct {
	CallId  string
	SSRC    string
	RtpIp   string
	SipNode string
	RtpPort string
	Time    string
}

func (s *Parser) getInviteLog() string {
	streamInfo := s.getIds()
	keywords := []string{"invite ok", streamInfo.GbId}
	if streamInfo.ChId != "" {
		keywords = append(keywords, streamInfo.ChId)
	}
	re := s.query(keywords)
	return s.fetchQvsServerLog(re)
}

func (s *Parser) parseInviteInfo(raw string) (inviteInfo InvitInfo, err error) {
	inviteInfo.CallId, err = s.getValByRegex(raw, `callId: (\d+)`)
	if err != nil {
		return
	}
	inviteInfo.SSRC, err = s.getValByRegex(raw, `ssrc:  (\d+)`)
	if err != nil {
		return
	}
	inviteInfo.RtpIp, err = s.getValByRegex(raw, `rtpAccessIp: (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	if err != nil {
		return
	}
	inviteInfo.SipNode, err = s.getValByRegex(raw, `rtpNode: (\S+)`)
	if err != nil {
		return
	}
	inviteInfo.RtpPort, err = s.getValByRegex(raw, `rtpPort: (\d+)`)
	if err != nil {
		return
	}
	inviteInfo.Time, err = s.getValByRegex(raw, `(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}.\d+)`)
	if err != nil {
		return
	}
	return
}

func (s *Parser) getRtpLog(taskId, nodeId, ssrc, chid string) (string, error) {
	re := fmt.Sprintf("%s.*got first|%s.*delete_channel|%s.*stream idle timeout|tcp attach.*%s|%s.*reset by peer", taskId, taskId, taskId, ssrc, chid)
	return s.searchLogs(nodeId, "qvs-rtp", re)
}

func (s *Parser) getSipLog(node, callid string) (string, error) {
	ss := strings.Split(node, "_")
	service := "qvs-sip"
	if len(ss) > 1 {
		service += ss[1]
	}
	re := fmt.Sprintf("response invite.*%s|INVITE response.*%s|sip_bye.*%s", callid, callid, callid)
	return s.searchLogs(ss[0], service, re)
}

func (s *Parser) getCreateChLogs(inviteTime, streamId, nodeId string) (string, error) {
	re := fmt.Sprintf("%s.*create_channel.*%s", inviteTime[:18], streamId)
	return s.searchLogs(nodeId, "qvs-rtp", re)
}

func (s *Parser) getRtpNode(rtpIp string) (string, error) {
	start := time.Now()
	re := fmt.Sprintf("/stream/publish/check.*%s", rtpIp)
	result, err := s.searchThemisd(re)
	if err != nil {
		return "", err
	}
	log.Println("get rtp node cost:", time.Since(start), "result:", result)
	return s.getValByRegex(result, `"nodeId":"(.*?)"`)
}

func (s *Parser) getInviteMsg(inviteInfo *InvitInfo, chid string) (string, error) {
	//streamInfo := s.getIds()
	//Keywords := []string{chid, inviteInfo.CallId, inviteInfo.RtpIp, "0" + inviteInfo.SSRC}
	Keywords := []string{chid, inviteInfo.CallId, "0" + inviteInfo.SSRC}
	query := s.multiLineQuery(Keywords)
	return s.searchLogsMultiLine(inviteInfo.SipNode, "qvs-sip", query)
}

func (s *Parser) getChidOfIPC(node, gbid string) (string, error) {
	Keywords := []string{"real chid of", gbid}
	query := s.query(Keywords)
	raw, err := s.searchLogsOne(node, "qvs-sip", query)
	if err != nil {
		return "", err
	}
	chid, err := s.getValByRegex(raw, `real chid of \d+ is (\d+)`)
	if err != nil {
		return "", err
	}
	//log.Println("chid:", chid)
	return chid, nil
}

func (s *Parser) getSipInviteRespLog(nodeid, callid string, resultChan chan<- string) {
	go func() {
		res, err := s.getSipLog(nodeid, callid)
		if err != nil {
			log.Println("get sip invite resp log err:", err)
			resultChan <- ""
			return
		}
		resultChan <- res
	}()
}

func (s *Parser) getStartStreamLog(inviteTime string) string {
	streamInfo := s.getIds()
	re := fmt.Sprintf("%s.*start a.*stream.*%s|devices/%s/start|rebuild strean.*%s.*%s", inviteTime[:18], s.Conf.StreamId, streamInfo.GbId, streamInfo.GbId, streamInfo.ChId)
	return s.fetchCenterAllServiceLogs(re)
}

func (s *Parser) pullFailLogAnalyse(final string) {
	if strings.Contains(final, "status:200") {
		log.Println("信令响应了200 ok")
	}
	if strings.Contains(final, "got first") {
		log.Println("设备有udp推流")
	} else if strings.Contains(final, "tcp attach") {
		log.Println("设备有tcp推流")
	} else {
		log.Println("设备没有推流")
	}
	if strings.Contains(final, "reset by") {
		log.Println("对端断开了tcp连接")
	}
	if strings.Contains(final, "sip_bye") {
		log.Println("有收到bye请求")
	}
}

/*
 * invite √
 * invite resp √
 * bye √
 * bye resp √
 * create channel √
 * delete channel √
 * got first √
 * tcp attach √
 * inner stean
 * catalog invite
 * reset by peer √
 * rtp 日志，过滤某个gbid
 * create channel 时间点之后的delete channel √
 * create channel时间点之后的idle timeout √
 */
func (s *Parser) streamPullFail() {
	final := ""
	streamInfo := s.getIds()
	raw := s.getInviteLog()
	final += raw
	inviteInfo, err := s.parseInviteInfo(raw)
	if err != nil {
		log.Println("get invite info err:", err)
		return
	}
	log.Println("callid:", inviteInfo.CallId, "ssrc:", inviteInfo.SSRC, "rtpIp:", inviteInfo.RtpIp, "sipNode:", inviteInfo.SipNode, "rtpPort:", inviteInfo.RtpPort, "time:", inviteInfo.Time)
	if streamInfo.ChId == "" {
		chid, err := s.getChidOfIPC(inviteInfo.SipNode, streamInfo.GbId)
		if err != nil {
			log.Fatalln(err)
		}
		streamInfo.ChId = chid
	}
	log.Println("real chid:", streamInfo.ChId)
	params := streamInfo.ChId + "," + inviteInfo.CallId
	dev, err := getDevice(streamInfo.GbId)
	if err != nil {
		log.Fatalln(err)
	}
	s.GetSipMsg(dev.NodeId, params)
	resultChan := make(chan string)
	s.getSipInviteRespLog(dev.NodeId, inviteInfo.CallId, resultChan)
	rtpNodeId, err := s.getNodeByIP(inviteInfo.RtpIp)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("rtpNodeId:", rtpNodeId)
	createChLog, err := s.getCreateChLog(inviteInfo.Time, rtpNodeId)
	if err != nil {
		log.Println(err)
		inviteRespLog := <-resultChan
		final += inviteRespLog
		if err := ioutil.WriteFile("out.log", []byte(final), 0644); err != nil {
			log.Fatalln(err)
		}
		return
	}
	final += createChLog
	createChTime, err := s.getValByRegex(createChLog, `(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}.\d+)`)
	if err != nil {
		createChTime, err = s.getValByRegex(createChLog, `\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\]`)
		if err != nil {
			log.Fatalln(err)
		}
	}
	//log.Println("createChLog:", createChLog)
	_, taskId, match := s.parseRtpLog(createChLog)
	if !match {
		log.Fatalln("get task id from create ch log err")
	}
	log.Println("taskId:", taskId)
	start2 := time.Now()
	rtpLog, err := s.getRtpLog(taskId, rtpNodeId, inviteInfo.SSRC, streamInfo.ChId)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("get rtp log cost:", time.Since(start2))
	//log.Println("rtpLog:", rtpLog)
	final += rtpLog
	delChLog, err := s.getDeleteChLog(createChTime, rtpNodeId)
	if err != nil {
		log.Println(err)
	}
	final += delChLog + "\n"
	//log.Println("delete ch log:", delChLog)
	inviteRespLog := <-resultChan
	final += inviteRespLog
	s.pullFailLogAnalyse(final)
	if err := ioutil.WriteFile("/tmp/streamPullFail.log", []byte(final), 0644); err != nil {
		log.Fatalln(err)
	}
}

func (s *Parser) splitSipMsg(raw string) []string {
	ss := strings.Split(raw, "\r\n")
	sipMsg := ""
	msgs := []string{}
	for _, str := range ss {
		if strings.Contains(str, "---") {
			if sipMsg == "" {
				continue
			}
			msgs = append(msgs, sipMsg)
			sipMsg = ""
			continue
		}
		sipMsg += str + "\r\n"
		if strings.Contains(str, "Content-Length") {
			sipMsg += "\r\n"
		}
	}
	return msgs
}

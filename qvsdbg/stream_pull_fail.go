package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
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

func (s *Parser) getInviteLog() (string, error) {
	streamInfo := s.getIds()
	keywords := []string{"invite ok", streamInfo.GbId}
	if streamInfo.ChId != "" {
		keywords = append(keywords, streamInfo.ChId)
	}
	query := s.query(keywords)
	resultChan := make(chan string)
	wg := sync.WaitGroup{}
	wg.Add(4)
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	for _, node := range nodes {
		go s.doSearch(node, "qvs-server", query, resultChan, &wg)
	}
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	var finalResult string
	for str := range resultChan {
		finalResult += str
	}
	logs, err := s.getNewestLog(finalResult)
	if err != nil {
		log.Println("get newest loggg err:", err)
		return "", err
	}
	//log.Println(node, raw)
	return logs, nil
}

func (s *Parser) getInviteInfo() (inviteInfo InvitInfo, err error) {
	raw, err := s.getInviteLog()
	if err != nil {
		return
	}
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

func (s *Parser) getRtpLog(taskId, nodeId, ssrc string) (string, error) {
	re := fmt.Sprintf("%s.*got first|%s.*delete_channel|%s.*stream idle timeout|tcp attach.*%s", taskId, taskId, taskId, ssrc)
	return s.searchLogs(nodeId, "qvs-rtp", re)
}

func (s *Parser) getCreateChLogs(inviteTime, streamId, nodeId string) (string, error) {
	re := fmt.Sprintf("%s.*create_channel.*%s", inviteTime[:18], streamId)
	return s.searchLogs(nodeId, "qvs-rtp", re)
}

func (s *Parser) getRtpNode(rtpIp string) (string, error) {
	start := time.Now()
	re := fmt.Sprintf("/stream/publish/check.*%s", rtpIp)
	result, err := searchThemisd(re)
	if err != nil {
		return "", err
	}
	log.Println("get rtp node cost:", time.Since(start))
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

/*
 * invite √
 * invite resp √
 * bye √
 * bye resp √
 * create channel √
 * delete channel √
 * channel remove
 * got first
 * tcp attach √
 * inner stean
 * catalog invite
 */
func (s *Parser) streamPullFail() {
	streamInfo := s.getIds()
	inviteInfo, err := s.getInviteInfo()
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
	start := time.Now()
	params := streamInfo.ChId + "," + inviteInfo.CallId
	sipMsgs, err := GetSipMsg(params)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println(sipMsgs)
	log.Println("cost:", time.Since(start))
	msgs := s.splitSipMsg(sipMsgs)
	log.Println("msgs: ", len(msgs))
	//log.Println(msgs)
	rtpNodeId, err := s.getRtpNode(inviteInfo.RtpIp)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("rtpNodeId:", rtpNodeId)
	createChLog, err := s.getCreateChLog(inviteInfo.Time, rtpNodeId)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("createChLog:", createChLog)
	_, taskId, match := s.parseRtpLog(createChLog)
	if !match {
		log.Fatalln("get task id from create ch log err")
	}
	log.Println("taskId:", taskId)
	rtpLog, err := s.getRtpLog(taskId, rtpNodeId, inviteInfo.SSRC)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("rtpLog:", rtpLog)
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

#!/usr/bin/python
# -*- coding: UTF-8 -*-

import sys
import json
import plyvel
import os
import time
import requests
import re
import logging as log

reload(sys)
sys.setdefaultencoding('utf8')

conf = "/usr/local/etc/pdr.conf"
logfile = "/tmp/sip_invite.log"

class Query:
    def __init__(self, gbid, chid):
        self.InviteReq = 'action=sip_invite&chid=' + chid + '&id=' + gbid
        self.InviteCheck = 'error device->invite sipid =' + chid + ' state:'
        self.H265 = 'gb28181 gbId ' + chid + ', ps map video es_type=h265'
        self.DeviceOffline = 'device ' + chid + ' offline'
        self.UdpRtp = 'gb28181 rtp enqueue : client_id ' + chid
        self.ResetByPeer = 'read() [src/protocol/srs_service_st.cpp:524][errno=104](Connection reset by peer)'
        streamId = ''
        if gbid != chid:
            streamId = gbid + '_' + chid 
        else:
            streamId = gbid
        self.streamId = streamId
        self.TcpAttach = 'gb28181: tcp attach new stream channel id:' + streamId
        self.InviteResp = 'gb28181 request client id=' + chid + ' response invite'
        self.InviteCheck = 'error device->invite sipid =' + chid + ' state:'
        if gbid == chid:
            self.IllegalSsrc = "ssrc illegal on tcp payload chaanellid:" + chid
        else:
            self.IllegalSsrc = "ssrc illegal on tcp payload chaanellid:" + gbid + "_" + chid
        self.CreateChannel = "id=" + streamId + "&port_mode=fixed"
        self.MediaInfo = "gb28181 gbId " + streamId + ", ps map video es_type="
        self.LostPkt = "gb28181: client_id " + streamId + " decode ps packet"

class Pdr:
    def __init__(self, query):
        token = self.getToken()
        self.baseUrl = 'http://qvs-pdr.qnlinking.com/api/v1/jobs'
        self.headers = {'content-type': 'application/json', 'Authorization': token[:len(token) - 1]}
        self.query = query

    def getToken(self):
        with open(conf, mode='r') as f:
            buf = f.read()
            f.close
            return buf

    def createJob(self, query, minus=60):
        curTs = int(round(time.time() * 1000))
        duration = int(minus)*60*1000
        startTime = str(curTs - duration)
        data = { "query" : query, "startTime": startTime, "endTime" : curTs}
        resp = requests.post(self.baseUrl, headers=self.headers, data=json.dumps(data))
        if resp.status_code != 200:
            log.info(resp.content)
            return None
        jres = json.loads(resp.content)
        return jres['id']

    def getJobInfo(self, jobId):
        url = self.baseUrl + "/" + jobId
        resp = requests.get(url, headers=self.headers)
        #print(resp.content)
        jres = json.loads(resp.content)
        return jres['process']

    def waitSearchDone(self, jobId):
        print("searching...")
        while True:
            process = self.getJobInfo(jobId)
            time.sleep(0.2)
            if process == 1:
                print("search job done")
                return

    def getLogs(self, jobId):
        url = self.baseUrl + "/" + jobId + "/events?rawLenLimit=false&pageSize=1000&order=desc&sort=updateTime"
        resp = requests.get(url, headers=self.headers)
        jres = json.loads(resp.content)
        #print(resp.content)
        return jres

    def getLog(self, query, minus=60):
        jobId = self.createJob(query, minus)
        if jobId is None:
            return None
        log.info('create job to get sip log, jobid: %s', jobId)
        self.waitSearchDone(jobId)
        logs = self.getLogs(jobId)
        rows = logs['rows']
        i = 0
        log.info("total: " + str(len(rows)))
        rawlog = ""
        rtpnode = ""
        while i < len(rows):
            raw = rows[i]['_raw']
            rawlog += raw['value'] + '\n'
            if rows[i]['host']['value'] != 'jjh1445':
                rtpnode = rows[i]['host']['value']
            i = i + 1
        return rawlog, rtpnode

    def saveFile(self, name, buf):
        with open(name, mode='w') as f:
            f.write(buf)
            f.close

    def wrapKeyword(self, keyword, isEnd = False):
        if isEnd:
            return '(\"' + keyword + '\")'
        else:
            return '(\"' + keyword + '\") or '

    def getPullStreamLog(self, minus=60):
        query = self.wrapKeyword(self.query.InviteReq) \
            + self.wrapKeyword(self.query.IllegalSsrc) \
            + self.wrapKeyword(self.query.DeviceOffline) \
            + self.wrapKeyword(self.query.InviteCheck) \
            + self.wrapKeyword(self.query.InviteResp) \
            + self.wrapKeyword(self.query.TcpAttach) \
            + self.wrapKeyword(self.query.UdpRtp) \
            + self.wrapKeyword(self.query.H265) \
            + self.wrapKeyword(self.query.LostPkt) \
            + self.wrapKeyword(self.query.CreateChannel) \
            + "\"rtmp connect ok url=rtmp\"*" + self.query.streamId + "* or " \
            + self.wrapKeyword(self.query.MediaInfo) \
            + self.wrapKeyword(self.query.ResetByPeer, True) \
            + ' repo=logs' + "| sort 1000000 by _time asc"
        log.info("query: %s", query)
        rawlog,rtpnode = self.getLog(query, minus)
        if rawlog is None:
            return None
        self.saveFile(logfile, rawlog)
        return rawlog.split('\n'), rtpnode

class Parser:
    def __init__(self, query, gbid, chid=""):
        self.query = query
        self.gbid = gbid
        self.chid = chid
        with open(logfile, 'r') as f:
            buf = f.read()
            self.lines = buf.split('\n')

    def getLogMeta(self, log):
        res = re.findall(r'[[](.*?)[]]', log)
        dateTime = res[0]
        taskId = res[3]
        return dateTime,taskId

    def searchLine(self, start, keyword, direction='forward'):
        end = len(self.lines)
        step = 1
        if direction == 'backword':
            step = -1
            #end = len(self.lines)
            end = 0
        for i in range(start, end, step):
            if keyword in self.lines[i]:
                return self.lines[i], i
        return None, None

    def getSsrc(self):
        line, i = self.searchLine(len(self.lines)-1, self.query.InviteReq, 'backword')
        if line is None:
            log.info('get invite req error')
            return
        pos = line.find('ssrc=')
        if pos == -1:
            log.info('get ssrc error')
            return
        end = line.find('&token=')
        if end == -1:
            log.info('find token error')
            return
        ssrc = line[pos+len('ssrc=') : end]
        return ssrc

    def getNodeIp(self):
        line, i = self.searchLine(len(self.lines)-1, self.query.InviteReq, 'backword')
        if line is None:
            log.info('get invite req error')
            return
        pos = line.find('ip=')
        if pos == -1:
            log.info('get node ip error')
            return
        end = line.find('&rtp_port=')
        if end == -1:
            log.info('find rtp_port error')
            return
        nodeIp = line[pos+len('ip=') : end]
        return nodeIp

    def tcpProc(self, line, num):
        log.info('有RTP OVER TCP过来了')
        dateTime, taskId = self.getLogMeta(line)
        tcpClose = '[' + taskId + ']:' + self.query.ResetByPeer 
        line, _num = self.searchLine(num, tcpClose, 'backword')
        if not line is None:
            log.info('摄像头断开TCP连接')
            return
        line, _num = self.searchLine(num, tcpClose, 'backword')
        if not line is None:
            log.info('视频流编码格式为H265')
            return

    def udpProc(self, line, num):
        log.info('收到RTP OVER UDP的包')

    def sipProc(self):
        line, num = self.searchLine(0, self.query.InviteResp)
        if line is None:
            log.info('[error] INVITE 信令没有收到response')
            return
        pos = line.find('status:')
        if pos == -1:
            log.info('[error] get invite status error')
            return -1
        dateTime, taskId = self.getLogMeta(line)
        code = line[pos+len('status:') : pos+len('status:')+3]
        log.info("%s invite %s resp: %s", dateTime, self.gbid ,code)

    def analysis(self):
        self.sipProc()
        line, num = self.searchLine(0, self.query.IllegalSsrc)
        if not line is None:
            log.info('设备发送过来的rtp包的ssrc非法')
            return
        line, num = self.searchLine(0, self.query.DeviceOffline)
        if not line is None:
            log.info('设备离线')
            return
        line, num = self.searchLine(0, self.query.TcpAttach)
        if not line is None:
            self.tcpProc(line, num)
            return
        line, num = self.searchLine(0, self.query.UdpRtp)
        if not line is None:
            self.udpProc(line, num)
            return
        log.info('[error] UDP和TCP都没有收到RTP包')
        ssrc = self.getSsrc()
        log.info('ssrc: %s', ssrc)
        nodeIp = self.getNodeIp()
        log.info('nodeIp: %s', nodeIp)


def main(gbid, chid, duration):
    query = Query(gbid, chid)
    pdr = Pdr(query)
    minus = 30
    if duration != "":
        minus = int(duration)
    raw, rtpnode = pdr.getPullStreamLog(minus)
    log.info('rtpnode: %s', rtpnode)
    parser = Parser(query, gbid)
    parser.analysis()

if __name__ == '__main__':
    log.basicConfig(level=log.INFO, format='%(filename)s:%(lineno)d: %(message)s')
    if len(sys.argv) < 3:
        log.info('args <gbid> <chid> [duration]')
        exit()
    main(sys.argv[1], sys.argv[2], sys.argv[3])


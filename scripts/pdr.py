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
gbid = ''
chid = ''
duration = 0

class Param:
    def __init__(self, gbid, chid):
        streamId = gbid
        if gbid != chid:
            streamId += '_' + chid
        self.streamId = streamId
        self.InviteReq = 'action=sip_invite&chid=' + chid + '&id=' + gbid
        self.InviteCheck = 'error device->invite sipid =' + chid + ' state:'
        self.H265 = 'gb28181 gbId ' + chid + ', ps map video es_type=h265'
        self.DeviceOffline = 'device ' + chid + ' offline'
        self.UdpRtp = 'gb28181 rtp enqueue : client_id ' + chid
        self.ResetByPeer = 'read() [src/protocol/srs_service_st.cpp:524][errno=104](Connection reset by peer)'
        self.TcpAttach = 'gb28181: tcp attach new stream channel id:' + streamId
        self.InviteResp = 'gb28181: INVITE response ' + chid + ' client status='
        self.InviteCheck = 'error device->invite sipid =' + chid + ' state:'
        self.IllegalSsrc = "ssrc illegal on tcp payload chaanellid:" + streamId 
        self.CreateChannel = "id=" + streamId + "&port_mode=fixed"
        self.MediaInfo = "gb28181 gbId " + streamId + ", ps map video es_type="
        self.LostPkt = "gb28181: client_id " + streamId + " decode ps packet"

param = Param("", "")

def wrapKeyword(keyword, isEnd = False):
    if isEnd:
        return '(\"' + keyword + '\")'
    else:
        return '(\"' + keyword + '\") or '

def saveFile(name, buf):
        with open(name, mode='w') as f:
            f.write(buf)
            f.close

def str2ts(str):
    timeArray = time.strptime(str, "%Y-%m-%d %H:%M:%S")
    timeStamp = int(time.mktime(timeArray))
    return timeStamp

def dumpStr(str):
    for i in str:
        print('%#x '%ord(i))

class Pdr:
    def __init__(self, query="", duration=5):
        token = self.getToken()
        self.baseUrl = 'http://qvs-pdr.qnlinking.com/api/v1/jobs'
        self.headers = {'content-type': 'application/json', 'Authorization': token[:len(token) - 1]}
        self.query = query
        self.duration = duration

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

    def fetchLog(self):
        jobId = self.createJob(self.query, self.duration)
        if jobId is None:
            return None
        #log.info('create job to get sip log, jobid: %s', jobId)
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

class Parser:
    def __init__(self, log=""):
        self.query = "" 
        self.log = log
        self.lines = log.split('\n')
        #with open(logfile, 'r') as f:
            #buf = f.read()
            #self.lines = buf.split('\n')

    def getLogMeta(self, log_):
        #log.info(log_)
        #dumpStr(log_)
        # ^[[0m[2021-09021]] 去除垃圾字符
        pos = log_.find('0m')
        new = log_
        if pos != -1:
            new = log_[pos+2:]
        res = re.findall(r'\[(.*?)\]', new)
        if len(res) == 0:
            log.info("[Error] get meta err:"+log_)
            return
        #log.info(res)
        dateTime = res[0]
        taskId = res[3]
        #log.info(res)
        return dateTime, taskId

    def getLatestLog(self, substr):
        logs = self.filterLog(substr)
        latestLog = ''
        latestTs = 0
        for line in logs:
            if line == '':
                continue
            date, taskId = self.getLogMeta(line)
            if date is None:
                continue
            #log.info(line)
            ts = str2ts(date[:len(date)-4]) # 时间的单位是精确到毫秒的
            if ts > latestTs:
                latestTs = ts
                latestLog = line
        #log.info("latestlog:"+latestLog)
        date, taskId = self.getLogMeta(latestLog)
        return {"date":date, "taskId":taskId, "raw":latestLog}
            
    # 过滤包含substr的所有字符串
    def filterLog(self, substr):
        ret = []
        for line in self.lines:
            if substr in line:
                ret.append(line)
        return ret


    def getValFromLog(self, log_, startKey, endKey):
        start = log_.find(startKey)
        if start == -1:
            return start
        end = log_.find(endKey)
        if end == -1:
            return end
        return log_[start+len(startKey):end]

    def getSSRC(self, log_):
        ssrc = self.getValFromLog(log_, "ssrc=", "&token")
        return ssrc

    def getNodeIp(self, log_):
        ip = self.getValFromLog(log_, "ip=", "&rtp_port=")
        return ip

def fetchLog():
    query = wrapKeyword(param.InviteReq) \
        + wrapKeyword(param.IllegalSsrc) \
        + wrapKeyword(param.DeviceOffline) \
        + wrapKeyword(param.InviteCheck) \
        + wrapKeyword(param.InviteResp) \
        + wrapKeyword(param.TcpAttach) \
        + wrapKeyword(param.UdpRtp) \
        + wrapKeyword(param.H265) \
        + wrapKeyword(param.LostPkt) \
        + wrapKeyword(param.CreateChannel) \
        + "\"rtmp connect ok url=rtmp\"*" + param.streamId + "* or " \
        + wrapKeyword(param.MediaInfo) \
        + wrapKeyword(param.ResetByPeer, True) \
        + ' repo=logs | sort 1000000 by _time asc'
    log.info("query: %s", query)
    pdr = Pdr(query, duration)
    rawlog, rtpnode = pdr.fetchLog()
    if rawlog is None:
        return None
    saveFile(logfile, rawlog)
    return rawlog, rtpnode

def run():
    raw, rtpNode = fetchLog()
    if raw is None:
        return
    log.info("rtpNode:"+rtpNode)
    parser = Parser(raw)
    #log.info(logs)
    ret = parser.getLatestLog(param.InviteReq)
    ssrc = parser.getSSRC(ret["raw"])
    ip = parser.getNodeIp(ret["raw"])
    log.info(ret["date"]+ ' ' + ret["taskId"] + " 请求invite,"+" ssrc: " + ssrc + ", rtpIp: "+ip)
    #log.info(ret)

# invite没有返回resp
# invite返回code非200
# 是否有tcp连接过来
# 是否有rtp over udp过来
# h265
# 设备离线
# illegal ssrc
# connection reset by peer
if __name__ == '__main__':
    log.basicConfig(level=log.INFO, format='%(filename)s:%(lineno)d: %(message)s')
    if len(sys.argv) < 3:
        log.info('args <gbid> <chid> [duration]')
        exit()
    gbid = sys.argv[1]
    chid = sys.argv[2]
    duration = sys.argv[3]
    param = Param(gbid, chid)
    run()


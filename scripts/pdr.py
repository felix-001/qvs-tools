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
        self.InviteReq = ['action=sip_invite&chid=' + chid + '&id=' + gbid]
        self.InviteCheck = ['error device->invite sipid =' + chid + ' state:']
        self.H265 = ['gb28181 gbId ' + chid + ', ps map video es_type=h265']
        self.DeviceOffline = ['device ' + chid + ' offline']
        self.UdpRtp = ['got first rtp pkt', streamId]
        self.ResetByPeer = ['read() [src/protocol/srs_service_st.cpp:524][errno=104](Connection reset by peer)']
        self.TcpAttach = ['gb28181: tcp attach new stream channel id:' + streamId]
        self.InviteResp = ['gb28181: INVITE response ' + chid + ' client status=']
        self.InviteCheck = ['error device->invite sipid =' + chid + ' state:']
        self.IllegalSsrc = ["ssrc illegal on tcp payload chaanellid:" + streamId]
        self.CreateChannel = ["id=" + streamId + "&is_audio_g711u"]
        self.MediaInfo = ["gb28181 gbId " + streamId + ", ps map video es_type="]
        self.LostPkt = ["gb28181: client_id " + streamId + " decode ps packet"]
        self.callIdQuery = ["after invite", gbid]
        self.rtmpConnect = ["rtmp connect ok url", streamId]
        self.realChid = ["real chid of " + gbid + " is"]

param = Param("", "")

def wrapKeyword(keywords, isEnd = False):
    query = ''
    for i in range(len(keywords)):
        query += '\"%s\"' % (keywords[i])
        if i < len(keywords) - 1:
            query += ' and '
    query = '(%s) ' % (query)
    if not isEnd:
        query += ' or ' 
    else:
        query += ' repo=logs | sort 1000000 by _time asc'
    return query

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
        #log.info("searching...")
        while True:
            process = self.getJobInfo(jobId)
            time.sleep(0.2)
            if process == 1:
                #log.info("search job done")
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
        #log.info("total logs: " + str(len(rows)))
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
        self.inviteReqTimeStamp = -1
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
        pos = log_.find('31m')
        if pos != -1:
            new = log_[pos+3:]
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
        if len(logs) == 0:
            return
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
        duration = latestTs - self.inviteReqTimeStamp
        # 日志如果和invite请求的日志时间差太多则认为是无效日志
        if self.inviteReqTimeStamp != -1 and duration > 20:
            return
        #log.info("latestlog:"+latestLog)
        date, taskId = self.getLogMeta(latestLog)
        return {"date":date, "taskId":taskId, "raw":latestLog, "duration":duration}
            
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

    def parseRealChid(self, log_):
        pos = log_.find('is ')
        if pos == -1:
            return
        return log_[pos+3:]

    def parseCallId(self, log_):
        pos = log_.find('callid:')
        if pos == -1:
            return
        return log_[pos+len('callid:'):]

    # invite请求
    def getInviteReq(self):
        ret = self.getLatestLog(param.InviteReq[0])
        if ret is None:
            log.info("[Error] 没有invite请求的日志")
            return
        self.inviteReqTimeStamp = str2ts(ret["date"][:-4])
        #log.info(self.inviteReqTimeStamp)
        self.ssrc = self.getSSRC(ret["raw"])
        self.rtpIp = self.getNodeIp(ret["raw"])
        log.info(ret["date"]+ ' ' + ret["taskId"] + " duration: 0 请求invite,"+" ssrc: " + self.ssrc + ", rtpIp: "+self.rtpIp)

    # 获取实际的chid
    def getRealChid(self):
        ret = self.getLatestLog(param.realChid[0])
        if ret is None:
            log.info("[Error] 没有获取实际chid的日志")
            return
        #log.info(ret)
        self.realChid = self.parseRealChid(ret["raw"])
        log.info(ret["date"]+ ' ' + ret["taskId"] + ' duration: ' + str(ret['duration']) + " 实际的chid: " + self.realChid)

    # 获取callid
    def getCallId(self):
        if not hasattr(self, 'realChid'):
            return
        callidStr = "after invite %s:%s ssrc:%s return callid:" % (gbid, self.realChid, self.ssrc)
        ret = self.getLatestLog(callidStr)
        if ret is None:
            log.info("[Error] 没有获取callid的日志")
            return
        self.callId = self.parseCallId(ret['raw'])
        #log.info(ret)
        log.info(ret["date"] + ' ' + ret["taskId"] + ' duration: ' + str(ret['duration']) + " callId: " + self.callId)

    # 创建rtp通道
    def getCreateChannel(self):
        ret = self.getLatestLog(param.CreateChannel[0])
        #log.info(ret)
        if ret is not None:
            log.info(ret["date"] + ' ' + ret["taskId"] + ' duration: ' + str(ret['duration']) + " 创建rtp通道")
        else:
            log.info("[Error] 没有创建rtp通道的日志")

    # 获取invite返回code
    def getInviteResp(self):
        if not hasattr(self, 'realChid'):
            return
        keywords = ["request client id=" + self.realChid, "status:", "callid="+self.callId]
        query = wrapKeyword(keywords, True)
        #log.info("query: %s", query)
        #log.info("fetch invite resp log")
        pdr = Pdr(query, duration)
        rawlog, rtpnode = pdr.fetchLog()
        if rawlog is None:
            log.info("没有获取到invite resp的日志")
            return None
        # 先暂存对象原来的lines，后面需要恢复
        tmp = self.lines
        self.lines = rawlog.split('\n')
        ret = self.getLatestLog("status:100")
        if ret is not None:
            log.info(ret["date"]+ ' ' + ret["taskId"] + ' duration: ' + str(ret['duration']) + " invite resp 100")
        else:
            log.info("[Error] 没有收到设备回复的100 Trying")
        ret = self.getLatestLog("status:200")
        if ret is not None:
            log.info(ret["date"]+ ' ' + ret["taskId"] + ' duration: ' + str(ret['duration']) + " invite resp 200")
        else:
            log.info("[Error] 没有收到设备回复的200 ok") 
        self.lines = tmp
        #log.info(rawlog)

    # tcp attach
    def getTcpAttach(self):
        ret = self.getLatestLog(param.TcpAttach[0])
        if ret is not None:
            if ret['duration'] < 0:
                log.info("[Error] 没有rtp over tcp连接过来")
                return
            log.info(ret["date"]+ ' ' + ret["taskId"] + ' duration: ' + str(ret['duration']) + " rtp over tcp 连接过来了")
        else:
            log.info("[Error] 没有rtp over tcp连接过来")
    
    # rtp over udp
    def getUdpRtp(self):
        ret = self.getLatestLog(param.UdpRtp[0])
        if ret is not None:
            log.info(ret["date"]+ ' ' + ret["taskId"] + " rtp over udp 数据包过来了")
        else:
            log.info("[Error] 没有收到rtp over udp的数据包")

    # h265
    def getH265(self):
        ret = self.getLatestLog(param.H265[0])
        if ret is not None:
            log.info(ret["date"]+ ' ' + ret["taskId"] + " 视频编码格式为h265")

    # illegal ssrc
    def getIllegalSSRC(self):
        ret = self.getLatestLog(param.IllegalSsrc[0])
        if ret is not None and ret['duration'] >= 0:
            log.info(ret["date"]+ ' ' + ret["taskId"] + ' duration: ' + str(ret['duration']) + " illegal ssrc")

    # connection reset by peer
    def getConnectionByPeer(self):
        ret = self.getLatestLog(param.ResetByPeer[0])
        if ret is not None:
            log.info(ret["date"]+ ' ' + ret["taskId"] + " 设备tcp连接过来之后又被设备关闭了,可能是由于平台发送了bye")
    
    # 设备离线
    def getDeviceOffline(self):
        ret = self.getLatestLog(param.DeviceOffline[0])
        if ret is not None:
            log.info(ret["date"]+ ' ' + ret["taskId"] + " 设备离线")


    def run(self):
        self.getInviteReq()
        self.getRealChid()
        self.getCallId()
        self.getCreateChannel()
        self.getInviteResp()
        self.getTcpAttach()
        self.getUdpRtp()
        self.getH265()
        self.getIllegalSSRC()
        self.getConnectionByPeer()
        self.getDeviceOffline()

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
        + wrapKeyword(param.rtmpConnect) \
        + wrapKeyword(param.MediaInfo) \
        + wrapKeyword(param.callIdQuery) \
        + wrapKeyword(param.realChid) \
        + wrapKeyword(param.ResetByPeer, True)
    log.info("query: %s", query)
    pdr = Pdr(query, duration)
    rawlog, rtpnode = pdr.fetchLog()
    if rawlog is None:
        return None
    saveFile(logfile, rawlog)
    return rawlog, rtpnode


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
    raw, rtpNode = fetchLog()
    if raw is None:
        sys.exit()
    log.info("rtpNode: "+rtpNode)
    parser = Parser(raw)
    parser.run()


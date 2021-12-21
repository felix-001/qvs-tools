#!/usr/bin/python
# -*- coding: UTF-8 -*-

import sys
import json
import os
import time
import requests
import re
import logging as log

reload(sys)
sys.setdefaultencoding('utf8')

def loadFile(file):
    with open(file, mode='r') as f:
            buf = f.read()
            f.close
            return buf

def getValFromLog(log_, startKey, endKey):
        start = log_.find(startKey)
        if start == -1:
            return start
        end = log_.find(endKey)
        if end == -1:
            return end
        return log_[start+len(startKey):end]

def getGbid(data):
    return getValFromLog(data, "client id=", " sip")

def httpGet(gbid):
    headers = {'Authorization': 'QiniuStub uid=0'}
    url = "http://10.20.76.42:7277/v1/devices?prefix=%s&qtype=0&line=20&offset=0" % (gbid)
    resp = requests.get(url, headers=headers)
    if resp.status_code != 200:
        print('http req err')
        return
    return json.loads(resp.content)

if __name__ == '__main__':
    data = loadFile('./data.log')
    rows = data.split('\n')
    gbids = []
    i = 0
    print('rows:' + str(len(rows)))
    while i < len(rows):
            gbid = getGbid(rows[i])
            if gbid not in gbids:
                gbids.append(gbid)
            i += 1
    print(gbids)
    print("total:"+str(len(gbids)))
    i = 0
    nonExistGbids = []
    existGbids = []
    while i < len(gbids):
        ret = httpGet(gbids[i])
        if len(ret['items']) == 0:
            nonExistGbids.append(gbids[i])
        else:
            existGbids.append(gbids[i])
	    i += 1
    print(nonExistGbids)
    print('non exist: ' + str(len(nonExistGbids)))
    i = 0
    s = ''
    while i < len(nonExistGbids):
        s += "AND (NOT %s)" % (gbids[i])
        i += 1
    print(existGbids)
    print('exist: ' + str(len(existGbids)))
    print(s)

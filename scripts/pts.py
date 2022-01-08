#!/usr/bin/python
# -*- coding: UTF-8 -*-
import os
import sys
import logging
import xml.etree.ElementTree as ET

# 1. 检查视频pts是否单调递增
# 2. 两帧之间的时间间隔是否正常

reload(sys)
sys.setdefaultencoding('utf8')

class Parser():
    def __init__(self, file):
        self.file = file
        self.ptsList = []

    def dump(self):
        cmd1 = 'ffprobe -show_frames -of csv ' + self.file + ' > /tmp/out.csv'
        os.popen(cmd1)
        cmd = 'ffprobe -show_frames -of xml ' + self.file
        res = os.popen(cmd).read()
        return res

    def ParseXml(self, xml, mediaType):
        root = ET.fromstring(xml)
        for child in root.iter('frame'):
            if child.get('media_type') == mediaType:
                self.ptsList.append(child.get('pkt_pts_time'))
        #print(self.ptsList)

    # 探测正常的duration, 所有的样本里面分布最多样本
    def probeNormalDur(self):
        durations = {}
        for i in range(len(self.ptsList)-1):
            duration = float(self.ptsList[i+1]) - float(self.ptsList[i])
            if durations.has_key(duration):
                durations[duration] += 1
            else:
                durations[duration] = 1
        max = 0
        for k in durations:
            if durations[k] > max:
                max = durations[k]
                self.normalDur = k 
                #print("max:"+str(max))
        print('正常的两帧之间的时间间隔:%f' % (self.normalDur))

    def printDurations(self):
        xml = self.dump()
        self.ParseXml(xml, 'video')
        self.probeNormalDur()
        total = 0
        max = 0
        min = 0
        for i in range(len(self.ptsList)-1):
            duration = float(self.ptsList[i+1]) - float(self.ptsList[i])
            if duration - self.normalDur > 0.02:
                print('第%d帧和前一帧的时间间隔过大, pts: %f dur: %f' % (i+1, float(self.ptsList[i+1]), duration))
                total += 1
                if duration > max:
                    max = duration
                if min == 0:
                    min = duration
                if duration < min:
                    min = duration

        print('总帧数: %d 总共异常: %d 异常率: %d%% 最大: %fs 最小: %fs' % (len(self.ptsList), total, float(total)/len(self.ptsList)*100, max, min))

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("args <videoFile>")
        sys.exit(0)

    parser = Parser(sys.argv[1])
    parser.printDurations()


#!/usr/bin/python
# -*- coding: UTF-8 -*-
import os
import sys
import logging
import xml.etree.ElementTree as ET

reload(sys)
sys.setdefaultencoding('utf8')

class Parser():
    def __init__(self, file):
        self.file = file

    def dump(self):
        cmd = 'ffprobe -show_frames -of xml ' + self.file
        res = os.popen(cmd).read()
        return res

    def ParseXml(self, xml, mediaType):
        ptsList = []
        root = ET.fromstring(xml)
        for child in root.iter('frame'):
            if child.get('media_type') == mediaType:
                ptsList.append(child.get('pkt_pts_time'))
        return ptsList

    def printDurations(self):
        xml = self.dump()
        ptsList = self.ParseXml(xml, 'video')
        durations = ""
        for i in range(len(ptsList)-1):
            duration = float(ptsList[i+1]) - float(ptsList[i])
            # round() 方法返回浮点数x的四舍五入值。
            durations += str(round(duration*1000, 2))+' '
        print(durations)


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("args <videoFile>")
        sys.exit(0)

    parser = Parser(sys.argv[1])
    parser.printDurations()


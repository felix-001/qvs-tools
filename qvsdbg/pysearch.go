package main

import (
	"encoding/base64"
	"fmt"
)

var PyMultiLineSearchScript = `
import sys
import os
import datetime

# -*- coding: UTF-8 -*-

keys = sys.argv[2].split(",")
keywords = {key: False for key in keys}

def checkKeywords(line):
        for key in keywords:
                if key in line:
                        keywords[key] = True
                        return

def allNeedFound():
        for val in keywords.values():
                if val == False:
                        return False
        return True

def resetKeywords():
	for key in keywords:
		keywords[key] = False

def multiLineSearch(file):
    logs = ""
    foundStart = False

    for line in file:
	if foundStart == False:
		if "---" in line:
			foundStart = True
			logs += line
		continue

	if "---" in line:
		if allNeedFound():
		    #print('all need found')
		    #logs += line
		    print(logs)
		    logs = line
		    resetKeywords()
		else:
		    #print('reset key words')
                    #logs = ""
                    logs = line
		    resetKeywords()
		continue
        checkKeywords(line)
        logs += line
    #print("not found")



directory = sys.argv[1]

start=datetime.datetime.now()
for root, dirs, files in os.walk(directory):
    for file_name in files:
        file_path = os.path.join(root, file_name)
        with open(file_path, "r") as file:
		#print('search', file_path)
	        multiLineSearch(file)
#print('cost', str(datetime.datetime.now()-start))
print('<--------------------------------------------------------------------------------------------------->')
`

func sshCmd(rawCmd, node string) string {
	jumpbox := "ssh -t liyuanquan@10.20.34.27"
	cmd := fmt.Sprintf("%s \"qssh %s \\\" %s \\\" \"", jumpbox, node, rawCmd)
	return cmd
}

func writeScriptToNode(node string) (string, error) {
	b64 := base64.StdEncoding.EncodeToString([]byte(PyMultiLineSearchScript))
	rawCmd := fmt.Sprintf("cd ~/liyq && echo %s | base64 -d > multiLineSearch.py", b64)
	cmd := sshCmd(rawCmd, node)
	return RunCmd(cmd)
}

func runMultiLineSearchScript(node string, args []string) (string, error) {
	rawCmd := "python /home/qboxserver/liyq/multiLineSearch.py "
	for _, arg := range args {
		rawCmd += arg + " "
	}
	cmd := sshCmd(rawCmd, node)
	//log.Println(cmd)
	return RunCmd(cmd)
}

func searchThemisd(re string) (string, error) {
	rawCmd := "cd /home/qboxserver/qvs-apigate/_package/run/auditlog/PILI-THEMISD/;"
	rawCmd += fmt.Sprintf("grep -E -h -m 1 \"%s\" * -R", re)
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	for _, node := range nodes {
		cmd := sshCmd(rawCmd, node)
		result, err := RunCmd(cmd)
		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}

	}
	return "", fmt.Errorf("themisd log not found")
}

var sep = "<--------------------------------------------------------------------------------------------------->\r\n"

// 参数列表，逗号分隔
// <chid>,<callid>,...
func GetSipMsg(params string) (string, error) {
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	result := ""
	for _, node := range nodes {
		if _, err := writeScriptToNode(node); err != nil {
			return "", err
		}
		args := []string{
			"/home/qboxserver/qvs-sip/_package/run/auditlog/sip_dump",
			params,
		}
		msg, err := runMultiLineSearchScript(node, args)
		if err != nil {
			return "", err
		}
		//log.Println("msg:", msg, "len:", len(msg))
		if msg == sep {
			continue
		}
		result += msg
	}
	return result, nil
}

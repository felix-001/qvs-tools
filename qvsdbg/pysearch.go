package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
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

func doRunCmd(cmd string, resultChan chan<- string, exitChan chan<- bool, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("start run cmd", cmd)
	result, err := RunCmd(cmd)
	if err != nil {
		log.Println("rum cmd", cmd, "err:", err)
		return
	}
	if result == "" {
		return
	}

	log.Println("run cmd", cmd, "done")
	resultChan <- result
	exitChan <- true // 发送退出信号
}

func searchThemisd(re string) (string, error) {
	rawCmd := "cd /home/qboxserver/qvs-apigate/_package/run/auditlog/PILI-THEMISD/;"
	rawCmd += fmt.Sprintf("grep -E -h -m 1 \"%s\" * -R", re)
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	resultChan := make(chan string)
	exitChan := make(chan bool) // 退出通道
	wg := sync.WaitGroup{}
	wg.Add(4)
	for _, node := range nodes {
		log.Println("search themisd node", node)
		cmd := sshCmd(rawCmd, node)
		go doRunCmd(cmd, resultChan, exitChan, &wg)
	}
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	done := false
	for !done {
		select {
		case str, ok := <-resultChan:
			if ok {
				return str, nil
			}
		case <-exitChan:
			done = true
		}
	}
	return "", fmt.Errorf("themisd log not found")
}

func runScript(node, params string, resultChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println("fetching sip log", node)
	if _, err := writeScriptToNode(node); err != nil {
		log.Println("write script err", node)
		return
	}
	args := []string{
		"/home/qboxserver/qvs-sip/_package/run/auditlog/sip_dump",
		params,
	}
	msg, err := runMultiLineSearchScript(node, args)
	if err != nil {
		log.Println("run script err", node, args)
		return
	}
	//log.Println("msg:", msg, "len:", len(msg))
	if msg == sep {
		return
	}
	log.Println("fetch sip log", node, "done")
	resultChan <- msg
}

var sep = "<--------------------------------------------------------------------------------------------------->\r\n"

// 参数列表，逗号分隔
// <chid>,<callid>,...
func GetSipMsg(params string) (string, error) {
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	resultChan := make(chan string)
	wg := sync.WaitGroup{}
	wg.Add(4)
	for _, node := range nodes {
		go runScript(node, params, resultChan, &wg)
	}
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	var finalResult string
	for str := range resultChan {
		finalResult += str
	}
	err := ioutil.WriteFile("out.sip", []byte(finalResult), 0644)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return finalResult, nil
}

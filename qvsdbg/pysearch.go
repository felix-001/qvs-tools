package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"
)

var PyMultiLineSearchScript = `
#!/usr/bin/python

import sys
import os
import datetime
import multiprocessing

# -*- coding: UTF-8 -*-

keys = sys.argv[1].split(",")
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

def multiLineSearch(file, results, filename):
    logs = ""
    foundStart = False
    result = ""

    for line in file:
        if foundStart == False:
            if "---" in line:
                foundStart = True
                logs += line
            continue

        if "---" in line:
            if allNeedFound():
                results.append(logs)
		#print(logs)
		result += logs
                logs = line
                resetKeywords()
            else:
                logs = line
                resetKeywords()
            continue
        checkKeywords(line)
        logs += line
    #print("result len is:"+str(len(result)))
    # skip <-------->
    if len(result) > 102:
    	write_to_file("/home/qboxserver/liyq/sip-search-result/"+filename, result)

def process_file(file_path, results, filename):
    #print("start search file " + file_path)
    #start = datetime.datetime.now()
    with open(file_path, "r") as file:
        multiLineSearch(file, results, filename)
    end = datetime.datetime.now()
    #print("task Time elapsed:" + str(end - start) + " " + file_path)

def write_to_file(file_path, content):
    with open(file_path, 'w') as file:
        file.write(content)

def get_directory_size(directory):
    total_size = 0

    for path, dirs, files in os.walk(directory):
        for file in files:
            filepath = os.path.join(path, file)
            total_size += os.path.getsize(filepath)

    return total_size

def print_file_contents(directory):
    for path, dirs, files in os.walk(directory):
        for file in files:
            filepath = os.path.join(path, file)

            with open(filepath, 'r') as f:
                contents = f.read()
		print(contents)
                #print(contents, filepath)

def delete_files(directory):
    for path, dirs, files in os.walk(directory):
        for file in files:
            filepath = os.path.join(path, file)
            os.remove(filepath)

if __name__ == '__main__':
    start = datetime.datetime.now()

    processes = []
    manager = multiprocessing.Manager()
    results = manager.list()

    paths = []
    for service in ["qvs-sip", "qvs-sip2", "qvs-sip3"]:
	paths.append("/home/qboxserver/"+service+"/_package/run/auditlog/sip_dump")

    delete_files("/home/qboxserver/liyq/sip-search-result")

    for directory in paths:
	#print(directory)
	for root, dirs, files in os.walk(directory):
		for file_name in files:
			if ".sip_raw" in file_name:
				# skip .sip_rawxxx.log.xxx
				continue
			file_path = os.path.join(root, file_name)
			process = multiprocessing.Process(target=process_file, args=(file_path, results, file_name))
			process.start()
			processes.append(process)

    for process in processes:
        process.join()

    #final = ""
    #for result in results:
    #    final += result
    #write_to_file("/tmp/out.txt", final)

    end = datetime.datetime.now()
    #print("cost time:" + str(end - start) )

    size = get_directory_size("/home/qboxserver/liyq/sip-search-result")

    if size > 5*1024*1024:
	print("err, search resutl too large, more than 5M, "+str(size))
	exit()

    print_file_contents("/home/qboxserver/liyq/sip-search-result")
`

var multiProcessSearch = `#!/usr/bin/python

import sys
import os
import datetime
import multiprocessing
import socket

def grep_search(directory, search_string):
    command = "LC_ALL=C grep -E -h '{0}' {1}* 2>/tmp/search-err.log".format(search_string, directory)
    if sys.argv[2] == "debug":
        print(command)
    start = datetime.datetime.now()
    output = os.popen(command).read()
    end = datetime.datetime.now()
    if sys.argv[2] == "debug":
        hostname = socket.gethostname()
        print("cost:"+str(end-start)+" " + command + " hostname: " + hostname)
    return output

def process_directory(directory, search_string, results):
    output = grep_search(directory, search_string)
    results.append(output)

if __name__ == '__main__':
    start = datetime.datetime.now()

    search_string = sys.argv[1]
    #services = ["qvs-server", "qvs-sip", "qvs-sip2", "qvs-sip3", "pili-themisd", "server-api", "themisd-api"]
    services = ["qvs-server", "qvs-sip", "qvs-sip2", "qvs-sip3", "server-api", "themisd-api"]
    if sys.argv[3] == "searchThemisd":
        services.append("pili-themisd")

    processes = []
    manager = multiprocessing.Manager()
    results = manager.list()
    node = os.popen("hostname").read()
    node = node[:len(node)-1]
    #print("node:"+node)

    for service in services:
        path = "/home/qboxserver/" + service + "/_package/run/"
        if service == "server-api":
            path = "/home/qboxserver/qvs-apigate/_package/run/auditlog/QVS-SERVER/"
        if service == "themisd-api":
            path = "/home/qboxserver/qvs-apigate/_package/run/auditlog/PILI-THEMISD"
            if node == "jjh1449":
                path += "-TEST"
            else:
                path += "/"
            if node == "bili-jjh9":
			#print("themisd api skip bili-jjh9")
                continue
        if service == "pili-themisd" and node == "bili-jjh9":
            continue

        process = multiprocessing.Process(target=process_directory, args=(path, search_string, results))
        process.start()
        processes.append(process)

    for process in processes:
        process.join()

    final = ""
    for result in results:
        final += result

    end = datetime.datetime.now()
    if sys.argv[2] == "debug":
        print("total cost"+str(end - start))
    print(final)
`

func sshCmd(rawCmd, node string) string {
	jumpbox := "ssh -t liyuanquan@10.20.34.27"
	cmd := fmt.Sprintf("%s \"qssh %s \\\"%s\\\"\"", jumpbox, node, rawCmd)
	return cmd
}

func writeScriptToNode(node string) (string, error) {
	b64 := base64.StdEncoding.EncodeToString([]byte(PyMultiLineSearchScript))
	rawCmd := fmt.Sprintf("cd ~/liyq && echo %s | base64 -d > multiLineSearch.py", b64)
	cmd := sshCmd(rawCmd, node)
	return RunCmd(cmd)
}

func writeServiceScriptToNode(node string) (string, error) {
	b64 := base64.StdEncoding.EncodeToString([]byte(multiProcessSearch))
	rawCmd := fmt.Sprintf("cd ~/liyq && echo %s | base64 -d > multi-process-search.py", b64)
	cmd := sshCmd(rawCmd, node)
	return RunCmd(cmd)
}

func runMultiLineSearchScript(node, query string) (string, error) {
	rawCmd := "python /home/qboxserver/liyq/multiLineSearch.py " + query
	cmd := sshCmd(rawCmd, node)
	//log.Println(cmd)
	return RunCmd(cmd)
}

func doRunCmd(cmd string, resultChan chan<- string, exitChan chan<- bool, wg *sync.WaitGroup) {
	defer wg.Done()
	//log.Println("start run cmd", cmd)
	result, err := RunCmd(cmd)
	if err != nil {
		log.Println("rum cmd", cmd, "err:", err)
		return
	}
	if result == "" {
		resultChan <- ""
		return
	}

	//log.Println("run cmd", cmd, "done")
	resultChan <- result
	exitChan <- true // 发送退出信号
}

func (s *Parser) searchThemisd(re string) (string, error) {
	nodes := []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
	resultChan := make(chan string)
	exitChan := make(chan bool) // 退出通道
	wg := sync.WaitGroup{}
	wg.Add(4)
	for _, node := range nodes {
		rawCmd := "cd /home/qboxserver/qvs-apigate/_package/run/auditlog/"
		if node == "jjh1449" || node == "bili-jjh9" {
			rawCmd += "PILI-THEMISD-TEST/;"
		} else {
			rawCmd += "PILI-THEMISD/;"
		}
		rawCmd += fmt.Sprintf("grep -E -h -m 1 '%s' * -R", re)
		//		log.Println("search themisd node", node)
		cmd := sshCmd(rawCmd, node)
		if s.Conf.Verbose {
			log.Println(cmd)
		}
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
	msg, err := runMultiLineSearchScript(node, params)
	if err != nil {
		log.Println("run script err", node, params)
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

var centerNodeList = []string{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}
var centerNodeList2 = []interface{}{"jjh1445", "jjh250", "jjh1449", "bili-jjh9"}

// 参数列表，逗号分隔
// <chid>,<callid>,...
func (s *Parser) GetSipMsg(node, params string) {
	//s.Conf.Node = node
	//s.Conf.Keywords = params
	go s.SearchSipLogs()
}

func doGetFileList(node string, resultChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	files, err := runSipLsCmd(node)
	if err != nil {
		log.Println("get sip raw file list err", node)
	}
	resultChan <- files
}

type Handler func(v interface{}) string

type ParaleelTask struct {
	Params  []interface{}
	Handler Handler
}

func NewParaleelTask(params []interface{}, handler Handler) *ParaleelTask {
	return &ParaleelTask{Params: params, Handler: handler}
}

func (p *ParaleelTask) doTask(param interface{}, resultChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	str := p.Handler(param)
	resultChan <- str
}

func (p *ParaleelTask) Run() string {
	resultChan := make(chan string)
	wg := sync.WaitGroup{}
	wg.Add(len(p.Params))
	for _, param := range p.Params {
		go p.doTask(param, resultChan, &wg)
	}
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	var finalResult string
	for str := range resultChan {
		finalResult += str
	}
	return finalResult
}

func getAllSipRawFiles() string {
	handler := func(v interface{}) string {
		node := v.(string)
		files, err := runSipLsCmd(node)
		if err != nil {
			log.Println("get sip raw file list err", node)
		}
		files = strings.TrimRight(files, "\r\n")
		fileList := strings.Split(files, "\r\n")
		result := ""
		for _, file := range fileList {
			result += node + "," + file + "\n"
		}
		return result
	}
	task := NewParaleelTask(centerNodeList2, handler)
	result := task.Run()
	return result
}

var sipLogPath = "/home/qboxserver/qvs-sip/_package/run/auditlog/sip_dump"

type SearchSipParam struct {
	Node  string
	File  string
	Query string
}

func (s *Parser) GetSipMsgs(query string) (string, error) {
	params := []interface{}{}
	for _, node := range centerNodeList {
		param := SearchSipParam{
			Node:  node,
			Query: query,
		}
		params = append(params, param)
	}
	handler := func(v interface{}) string {
		param := v.(SearchSipParam)
		log.Println("fetching sip log", param.Node, param.Node)
		if s.Conf.WritePyToNode {
			if str, err := writeScriptToNode(param.Node); err != nil {
				log.Println("write script err", param.Node, err, str)
				return ""
			}
		}
		msg, err := runMultiLineSearchScript(param.Node, query)
		if err != nil {
			log.Println("run script err", param.Node, query, err, msg)
			return ""
		}
		//log.Println("msg:", msg, "len:", len(msg))
		if msg == sep {
			return ""
		}
		log.Println("fetch sip log", param.Node, "done")
		return msg
	}
	task := NewParaleelTask(params, handler)
	result := task.Run()
	return result, nil
}

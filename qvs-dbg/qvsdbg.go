package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	adminIP      string
	gbID         string
	reqID        string
	localLogPath string
	refetchLog   bool
)

type Context struct {
	NsId       string
	SipNodeID  string
	SipSrvIP   string
	ProcessIdx string
}

func NewContext() *Context {
	return &Context{}
}

func httpReq(method, addr, body string, headers map[string]string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if resp.StatusCode != 200 {
		log.Println("status code", resp.StatusCode, string(resp_body), addr)
		return "", errors.New("http resp code err")
	}
	return string(resp_body), err
}

func qvsReq(method, addr, body string) (string, error) {
	headers := map[string]string{"authorization": "QiniuStub uid=0"}
	return httpReq(method, addr, body, headers)
}

func qvsGet(addr string) (string, error) {
	return qvsReq("GET", addr, "")
}

func qvsPost(addr, body string) (string, error) {
	headers := map[string]string{
		"authorization": "QiniuStub uid=0",
		"Content-Type":  "application/json",
	}
	return httpReq("POST", addr, body, headers)
}

type Device struct {
	Vendor   string `json:"vendor"`
	RemoteIP string `json:"remoteIp"`
	NsID     string `json:"nsId"`
	NodeID   string `json:"nodeId"`
}

func (c *Context) getNsID() (string, error) {
	path := fmt.Sprintf("devices?prefix=%s", gbID)
	uri := c.adminUri(path)
	resp, err := qvsGet(uri)
	if err != nil {
		return "", err
	}
	//log.Println(resp)
	data := struct {
		Devices []Device `json:"items"`
	}{}
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		log.Println(err)
		return "", err
	}
	if len(data.Devices) == 0 {
		return "", fmt.Errorf("device not found")
	}
	c.NsId = data.Devices[0].NsID
	log.Println("nsId:", c.NsId)
	return data.Devices[0].NsID, nil
}

func (c *Context) adminUri(path string) string {
	return fmt.Sprintf("http://%s:7277/v1/%s", adminIP, path)
}

func (c *Context) getSipNodeID() (string, error) {
	path := fmt.Sprintf("namespaces/%s/devices/%s", c.NsId, gbID)
	uri := c.adminUri(path)
	resp, err := qvsGet(uri)
	if err != nil {
		return "", err
	}
	var device Device
	if err := json.Unmarshal([]byte(resp), &device); err != nil {
		log.Println(err)
		return "", err
	}
	sipNodeId := device.NodeID
	processIdx := ""
	ss := strings.Split(device.NodeID, "_")
	if len(ss) == 2 {
		sipNodeId = ss[0]
		processIdx = ss[1]
	}
	c.SipNodeID = sipNodeId
	c.ProcessIdx = processIdx
	log.Println("sip nodeid:", c.SipNodeID)
	return device.NodeID, nil
}

func runCmd(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	if err != nil {
		//TODO 待优化
		return string(b), err
	}
	return string(b), nil
}

func (c *Context) deleteOldLogs() error {
	log.Println("clear old logs")
	if _, err := runCmd("rm -rf ~/logs/*"); err != nil {
		return err
	}
	return nil
}

func findDestStr(src string) (string, error) {
	compileRegex := regexp.MustCompile("[(](.*?)[)]")
	matchArr := compileRegex.FindStringSubmatch(src)
	if len(matchArr) > 0 {
		return matchArr[len(matchArr)-1], nil
	}
	return "", fmt.Errorf("parse err")
}

func (c *Context) getSipSrvIP() (string, error) {
	cmd := fmt.Sprintf("ping -c 1 %s", c.SipNodeID)
	res, _ := runCmd(cmd)
	ip, err := findDestStr(res)
	if err != nil {
		return "", err
	}
	c.SipSrvIP = ip
	log.Println("sip srv ip:", c.SipSrvIP)
	return ip, nil
}

func (c *Context) getSSRC(reqId string) (string, error) {
	return "", nil
}

func parseConsole() error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.StringVar(&reqID, "r", "", "input reqid")
	flag.StringVar(&adminIP, "i", "", "input admin ip")
	flag.StringVar(&gbID, "g", "", "input gbid")
	flag.BoolVar(&refetchLog, "f", false, "删除本地缓存的日志文件，重新从节点获取,否则直接读取本地缓存的日志")
	flag.StringVar(&localLogPath, "l", "/home/liyuanquan/logs", "本地缓存日志文件的路径")
	flag.Parse()
	if reqID == "" {
		return fmt.Errorf("please input reqId")
	}
	if adminIP == "" {
		return fmt.Errorf("please input adminip")
	}
	if gbID == "" {
		return fmt.Errorf("please input gbid")
	}
	return nil
}

func (c *Context) fetchLogs(srvName, nodeId, filenamePrefix string) error {
	if !refetchLog {
		return nil
	}
	log.Printf("start to fetch %s logs from %s\n", srvName, nodeId)
	start := time.Now().Unix()
	cmdstr := fmt.Sprintf("qscp qboxserver@%s:/home/qboxserver/%s/_package/run/%s.log* %s", nodeId, srvName, filenamePrefix, localLogPath)
	log.Println(cmdstr)
	if _, err := runCmd(cmdstr); err != nil {
		return err
	}
	log.Println("fetch", srvName, "logs from", nodeId, "done, cost time:", time.Now().Unix()-start, "service name:", srvName)
	return nil
}

func (c *Context) fetchSipLogs() error {
	srvName := "qvs-sip"
	if c.ProcessIdx != "" {
		srvName += c.ProcessIdx
	}
	return c.fetchLogs(srvName, c.SipNodeID, srvName)
}

func (c *Context) fetchRtpLogs(rtpPort, rtpNodeId string) error {
	srvName := "qvs-rtp"
	filenamePrefix := srvName + "_" + rtpPort[2:]
	return c.fetchLogs(srvName, rtpNodeId, filenamePrefix)
}

func (c *Context) getDatetimeFromFileName(file string) (string, error) {
	ss := strings.Split(file, "-")
	if len(ss) != 3 {
		return "", fmt.Errorf("split log file name err %v", []byte(file))
	}
	return ss[2], nil
}

// 从节点下载下来的日志文件可能有多个，这里需要获取到最新的
func (c *Context) getLatestLogFile(srvName string) (string, error) {
	files, err := ioutil.ReadDir(localLogPath)
	if err != nil {
		return "", err
	}
	latest := ""
	for _, file := range files {
		if !strings.Contains(file.Name(), srvName) {
			continue
		}
		if latest == "" {
			latest = file.Name()
			continue
		}
		latestLogfileTime, err := c.getDatetimeFromFileName(latest)
		if err != nil {
			return "", err
		}
		curLogfileTime, err := c.getDatetimeFromFileName(file.Name())
		if err != nil {
			return "", err
		}
		if strings.Compare(curLogfileTime, latestLogfileTime) > 0 {
			latest = file.Name()
		}
	}
	return latest, nil
}

func (c *Context) getLatestRtpLogFile(rtpNodeId, rtpPort string) (string, error) {
	srvName := "qvs-rtp"
	filenamePrefix := srvName + "_" + rtpPort[2:]
	cmdstr := fmt.Sprintf("qssh %s \"ls /home/qboxserver/%s/_package/run/%s.log*\"", rtpNodeId, srvName, filenamePrefix)
	// "qssh zz788 \"ls /home/qboxserver/qvs-rtp/_package/run/qvs-rtp_02*\""
	res, err := runCmd(cmdstr)
	if err != nil {
		return "", err
	}
	log.Printf("rtp log file list:\n%s", res)
	latest := ""
	scanner := bufio.NewScanner(strings.NewReader(res))
	for scanner.Scan() {
		line := scanner.Text()
		_, file := filepath.Split(line)
		if latest == "" {
			latest = file
			continue
		}
		if latest == "\n" {
			continue
		}
		latestLogfileTime, err := c.getDatetimeFromFileName(latest)
		if err != nil {
			return "", err
		}
		curLogfileTime, err := c.getDatetimeFromFileName(file)
		if err != nil {
			return "", err
		}
		if strings.Compare(curLogfileTime, latestLogfileTime) > 0 {
			latest = file
		}
	}
	return latest, nil
}

// 从文本里面，找到符合关键字列表的行
func (c *Context) findLineWithKeywords(file string, keywords []string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	lines := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		found := true
		for _, keyword := range keywords {
			if !strings.Contains(line, keyword) {
				found = false
				break
			}
		}
		if found {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("line not found")
	}
	return lines, nil
}

// s="hello&test=value&workd"
// start="hello&""
// end="&world"
// result = "value"
func (c *Context) getValByStartEndKeyword(s, startKeyword, endKeyword string) (string, error) {
	start := strings.Index(s, startKeyword)
	if start == -1 {
		return "", fmt.Errorf("keyword %s not found in %s", startKeyword, s)
	}
	snew := s[start+len(startKeyword):]
	if endKeyword == "" {
		return snew, nil
	}
	end := strings.Index(snew, endKeyword)
	if end == -1 {
		return "", fmt.Errorf("keyword %s not found in %s", endKeyword, s)
	}
	return snew[:end], nil
}

type QueryNodesOpt struct {
	IP string `json:"ip"`
}

type NodeInfo struct {
	ID string `json:"id"`
}

func (c *Context) getNodeIDByIP(ip string) (string, error) {
	query := QueryNodesOpt{IP: ip}
	jsonbody, err := json.Marshal(&query)
	if err != nil {
		return "", err
	}
	// TOOD: 4981从控制台传入
	uri := fmt.Sprintf("http://%s:4981/listnodes/basicinfo", adminIP)
	resp, err := qvsPost(uri, string(jsonbody))
	if err != nil {
		return "", err
	}
	data := &struct {
		Items []NodeInfo `json:"items"`
	}{}
	if err := json.Unmarshal([]byte(resp), data); err != nil {
		return "", err
	}
	return data.Items[0].ID, nil
}

/*
1. 拉流失败
	1.1 实时流
	1.2 历史流
2. 设备离线
3. 对讲失败
4. 云端录制失败
5. 如果通过admin获取不到nodeId，则通过pdr去获取
^[[31m[2023-01-17 11:33:57.440][Error][56736][d4262t5f][11] ssrc illegal on tcp payload chaanellid: ssrc :123116795 rtp_pack_len:8012
buf:13142 payload_type:96 peer_ip:120.193.152.166:26564 fd:39(Resource temporarily unavailable)
6. ssrc不正确的问题
7. nvr chid
8. /home/liyuanquan自动获取
9. 获取一些指标，比如有多少条sip 503， sip session remove， 带宽， illegal ssrc
10. 通过请求admin，请求启动拉流，sleep几秒钟，然后获取reqId
11. 响应经过时间
*/

/*
使用姿势:
./qvsdbg -i <adminIP> -r <reqId> -g <gbId> [-f]
*/

func main() {
	if err := parseConsole(); err != nil {
		fmt.Println(err)
		fmt.Println("usage: ./qvsdbg -i <adminIP> -r <reqId> -g <gbId> [-f]")
		flag.PrintDefaults()
		return
	}
	ctx := NewContext()
	if _, err := ctx.getNsID(); err != nil {
		log.Println(err)
		return
	}
	if _, err := ctx.getSipNodeID(); err != nil {
		log.Println(err)
		return
	}
	if _, err := ctx.getSipSrvIP(); err != nil {
		log.Println(err)
		return
	}
	if refetchLog {
		if err := ctx.deleteOldLogs(); err != nil {
			log.Println(err)
			return
		}
	}
	if err := ctx.fetchSipLogs(); err != nil {
		log.Println(err)
		return
	}
	latestSipLogFile, err := ctx.getLatestLogFile("qvs-sip")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("lastest sip log file:", latestSipLogFile)
	sipLogFile := fmt.Sprintf("/home/liyuanquan/logs/%s", latestSipLogFile)
	line, err := ctx.findLineWithKeywords(sipLogFile, []string{"xwMAAMw4YS5mXjsX", "31011500991320021468"})
	if err != nil {
		log.Println(err)
		return
	}
	//log.Println(line)
	ssrc, err := ctx.getValByStartEndKeyword(line[0], "ssrc=", "&talk_model")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("ssrc:", ssrc)
	callidLine, err := ctx.findLineWithKeywords(sipLogFile, []string{ssrc, "return callid"})
	if err != nil {
		log.Println(err)
		return
	}
	//log.Println("callid line:", callidLine)
	callid, err := ctx.getValByStartEndKeyword(callidLine[0], "return callid:", "")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("callid:", callid)
	inviteRespLine, err := ctx.findLineWithKeywords(sipLogFile, []string{callid, "respone method=INVITE"})
	if err != nil {
		log.Println(err)
		return
	}
	//log.Println("invite resp line:", inviteRespLine)
	inviteRespCode, err := ctx.getValByStartEndKeyword(inviteRespLine[0], "status:", " callid")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("invite resp code 0:", inviteRespCode)
	inviteRespCode1, err := ctx.getValByStartEndKeyword(inviteRespLine[1], "status:", " callid")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("invite resp code 1:", inviteRespCode1)
	rtpIp, err := ctx.getValByStartEndKeyword(line[0], "ip=", "&reqId")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("rtpIp:", rtpIp)
	rtpPort, err := ctx.getValByStartEndKeyword(line[0], "rtp_port=", "&rtp_proto")
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("rtpPort:", rtpPort)
	if len(rtpPort) != 4 {
		log.Println("invalid rtp port", rtpPort)
		return
	}
	rtpNodeId, err := ctx.getNodeIDByIP(rtpIp)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("rtp node id:", rtpNodeId)
	if err := ctx.fetchRtpLogs(rtpPort, rtpNodeId); err != nil {
		log.Println(err)
		return
	}
	latestRtpLogFile, err := ctx.getLatestRtpLogFile(rtpNodeId, rtpPort)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("latest rtp log file:", latestRtpLogFile)

}

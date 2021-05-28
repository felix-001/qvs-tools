package main

import (
	"flag"
	"log"
	"os/exec"
)

type ConsoleParams struct {
	logPath string
}

func Exec(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		return "", err
	}
	return string(b), err
}

func GetLatestLogFile(path string) (string, error) {
	cmdstr := "ls -t " + path + "qvs-log.log* | head -n 1"
	return Exec(cmdstr)
}

type LogParser struct {
	logFile string
	gbId    string
	chId    string
}

func NewLogParser(logFile string) *LogParser {
	return &LogParser{logFile: logFile}
}

type InviteInfo struct {
	ssrc  int
	rtpIp string
	time  string
}

func (self *LogParser) getLastInviteLog() (string, error) {
	cmdstr := "tac " + self.logFile + " | grep \"sip_invite&chid=" + self.chId + "&id=" + self.gbId + "-m 1"
	return Exec(cmdstr)
}

func (self *LogParser) GetInviteInfo() *InviteInfo {

}

func main() {
	log.SetFlags(log.Lshortfile)
	logPath := flag.String("logpath", "~/qvs-sip/_package/run/", "log file path")
	flag.Parse()
	params := &ConsoleParams{}
	params.logPath = *logPath
	log.Println(params.logPath)
	logFile, _ := GetLatestLogFile(params.logPath)
	log.Println(logFile)
	parser := NewLogParser(logFile)
	inviteInfo := parser.GetInviteInfo()
	log.Printf("%+v\n", inviteInfo)
}

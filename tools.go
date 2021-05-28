package main

import (
	"flag"
	"log"
	"os/exec"
)

type ConsoleParams struct {
	logPath string
}

func GetLatestLogFile(path string) (string, error) {
	cmdstr := "ls -t " + path + "qvs-log.log* | head -n 1"
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		return "", err
	}
	return string(b), err
}

func main() {
	log.SetFlags(log.Lshortfile)
	logPath := flag.String("logpath", "~/qvs-sip/_package/run/", "log file path")
	flag.Parse()
	params := &ConsoleParams{}
	params.logPath = *logPath
	log.Println(params.logPath)
	res, _ := GetLatestLogFile(params.logPath)
	log.Println(res)
}

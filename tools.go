package main

import (
	"flag"
	"log"
	"os/exec"
)

type ConsoleParams struct {
	logPath string
}

func GetLatestLogFile() (string, error) {
	cmd := exec.Command("ls", "-a", "-l", "-h")
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func main() {
	log.SetFlags(log.Lshortfile)
	logPath := flag.String("logpath", "~/qvs-sip/_package/run/", "log file path")
	flag.Parse()
	params := &ConsoleParams{}
	params.logPath = *logPath
	log.Println(params.logPath)
	res, _ := GetLatestLogFile()
	log.Println(res)
}

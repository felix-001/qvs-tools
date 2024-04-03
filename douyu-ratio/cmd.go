package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func RunCmd(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	//fmt.Println(cmd)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return string(b), err
	}
	raw := string(b)
	//log.Println(raw)
	if strings.Contains(raw, "Pseudo-terminal") || strings.Contains(raw, "Redirected to slot") {
		new := ""
		ss := strings.Split(raw, "\n")
		if len(ss) == 1 {
			return "", nil
		}
		for _, str := range ss {
			if strings.Contains(str, "Pseudo-terminal") || strings.Contains(str, "Redirected to slot") {
				continue
			}
			if len(str) == 0 {
				continue
			}
			new += str + "\r\n"
		}
		return new, nil
	}
	return raw, nil
}

func JumpboxCmd(rawCmd string) (string, error) {
	jumpbox := "ssh -t liyuanquan@10.20.34.27"
	cmd := fmt.Sprintf("%s \" %s \"", jumpbox, rawCmd)
	return RunCmd(cmd)
}

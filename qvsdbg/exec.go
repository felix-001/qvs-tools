package main

import (
	"os/exec"
	"strings"
)

func RunCmd(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	//cmd.Stderr = os.Stderr
	b, err := cmd.CombinedOutput()
	if err != nil {
		return string(b), err
	}
	//return string(b), nil
	raw := string(b)
	if strings.Contains(raw, "Pseudo-terminal") {
		new := ""
		ss := strings.Split(raw, "\n")
		for _, str := range ss {
			if strings.Contains(str, "Pseudo-terminal") {
				continue
			}
			new += str
		}
		return new, nil
	}
	return raw, nil
}

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
	//log.Println(raw)
	if strings.Contains(raw, "Pseudo-terminal") {
		new := ""
		ss := strings.Split(raw, "\n")
		if len(ss) == 1 {
			return "", nil
		}
		for _, str := range ss {
			if strings.Contains(str, "Pseudo-terminal") {
				continue
			}
			if len(str) == 0 {
				continue
			}
			//log.Println("str len:", len(str))
			new += str + "\r\n"
		}
		//log.Println("new:", new)
		return new, nil
	}
	return raw, nil
}

func RunPyCmd(cmdstr string, args []string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	//cmd.Args = args
	b, err := cmd.CombinedOutput()
	if err != nil {
		return string(b), err
	}
	//return string(b), nil
	raw := string(b)
	return raw, nil
}

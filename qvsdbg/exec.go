package main

import "os/exec"

func RunCmd(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return string(b), err
	}
	return string(b), nil
}

package main

import (
	"fmt"
	"log"
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

func (s *Parser) searchLogs(node, service, re string) (string, error) {
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/%s/_package/run;grep -E -h '%s' * -R\\\"\"", node, service, re)
	if s.Conf.Verbose {
		log.Println(cmd)
	}
	return RunCmd(cmd)
}

func (s *Parser) searchApiLog(node, service, re string) (string, error) {
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/qvs-apigate/_package/run/auditlog/%s;grep -E -h '%s' * -R\\\"\"", node, service, re)
	if s.Conf.Verbose {
		log.Println(cmd)
	}
	return RunCmd(cmd)
}

// 遇到一个匹配的就停止
func (s *Parser) searchLogsOne(node, service, re string) (string, error) {
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/%s/_package/run;grep -E -h -m 1 '%s' * -R \\\"\"", node, service, re)
	if s.Conf.Verbose {
		log.Println(cmd)
	}
	return RunCmd(cmd)
}

func qsshCmd(rawCmd, node string) string {
	jumpbox := "ssh -t liyuanquan@10.20.34.27"
	cmd := fmt.Sprintf("%s \"qssh %s \\\" %s \\\" \"", jumpbox, node, rawCmd)
	return cmd
}

func grepCmd(srcFile, node, re string) string {
	//srcFile := fmt.Sprintf("/home/qboxserver/%s/_package/run")
	grepCmd := fmt.Sprintf("grep -E -h '%s' %s -R", re, srcFile)
	return qsshCmd(grepCmd, node)
}

func runLsCmd(rawCmd, node string) (string, error) {
	//rawCmd := fmt.Sprintf("ls /home/qboxserver/%s/_package/run", service)
	cmd := qsshCmd(rawCmd, node)
	return RunCmd(cmd)
}

func runServiceLsCmd(service, node string) (string, error) {
	rawCmd := fmt.Sprintf("ls /home/qboxserver/%s/_package/run/*.log*", service)
	result, err := runLsCmd(rawCmd, node)
	if err != nil {
		return "", err
	}
	result = strings.TrimRight(result, "\r\n")
	return result, nil
}

func runSipLsCmd(node string) (string, error) {
	rawCmd := "ls /home/qboxserver/qvs-sip/_package/run/auditlog/sip_dump/*.log*"
	result, err := runLsCmd(rawCmd, node)
	if err != nil {
		return "", err
	}
	//result = strings.TrimRight(result, "\r\n")
	return result, nil
}

func searchServiceLog(node, re string) (string, error) {
	rawCmd := "ls /home/qboxserver/qvs-sip/_package/run/auditlog/sip_dump/*.log*"
	result, err := runLsCmd(rawCmd, node)
	if err != nil {
		return "", err
	}
	result = strings.TrimRight(result, "\r\n")
	return result, nil
}

func (s *Parser) searchLogsMultiLine(node, service, re string) (string, error) {
	cmd := fmt.Sprintf("ssh -t liyuanquan@10.20.34.27 \"qssh %s \\\"cd /home/qboxserver/%s/_package/run/auditlog/sip_dump;grep -h -Pzo '%s' * -R\\\"\"", node, service, re)
	log.Println(cmd)
	return RunCmd(cmd)
}

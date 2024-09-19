package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
)

func (s *Parser) dySecret() (string, string) {
	t, err := str2time("2034-12-01 00:00:00")
	if err != nil {
		s.logger.Error().Err(err).Msg("conv time err")
		return "", ""
	}
	wsTime := fmt.Sprintf("%x", t.Unix())
	raw := s.conf.Secret + s.conf.Stream + wsTime
	hash := md5.Sum([]byte(raw))
	wsSecret := hex.EncodeToString([]byte(hash[:]))
	return wsTime, wsSecret
}

func (s *Parser) DyPlay() {
	//pcdn := s.getPcdn()
	if s.conf.Pcdn == "" {
		_, s.conf.Pcdn = s.getPcdnFromSchedAPI(true, false)
	}
	wsTime, wsSecret := s.dySecret()
	cmdStr := fmt.Sprintf("./xs -addr %s -path %s/%s.xs -q \"wsSecret=%s&wsTime=%s&domain=qn-ss.douyucdn.cn&sourceID=2594498916\" -f out.xs",
		s.conf.Pcdn, s.conf.Bucket, s.conf.Stream, wsSecret, wsTime)
	log.Println("cmd:", cmdStr)
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v %s %s\n", err, string(output), cmdStr)
		return
	}
	fmt.Println(string(output))

}

func (s *Parser) DyPcdn() {
	fmt.Println(s.getPcdnFromSchedAPI(true, false))
}

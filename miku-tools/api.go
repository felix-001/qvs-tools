package main

import (
	"fmt"
	"middle-source-analysis/util"
)

func (s *Parser) GetDomain() {
	addr := fmt.Sprintf("http://%s.mls.cn-east-1.qiniumiku.com/?domainConfig&name=%s", s.conf.Bucket, s.conf.Domain)
	resp, err := util.S3get(addr, s.conf)
	if err != nil {
		s.logger.Error().Err(err).Str("addr", addr).Msg("UpdateDomain")
		return
	}
	fmt.Println(resp)
}

func (s *Parser) UpdateDomain() {
	addr := fmt.Sprintf("http://%s.mls.cn-east-1.qiniumiku.com/?domainConfig&name=%s", s.conf.Bucket, s.conf.Domain)
	body := `{"streamConf": {"enableXsStream": true}}`
	resp, err := util.S3patch(addr, body, s.conf)
	if err != nil {
		s.logger.Error().Err(err).Str("addr", addr).Msg("UpdateDomain")
		return
	}
	fmt.Println(resp)
}

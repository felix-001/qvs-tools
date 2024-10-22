package main

import "fmt"

func (s *Parser) GetDomain() {
	addr := fmt.Sprintf("http://%s.mls.cn-east-1.qiniumiku.com/?domainConfig&name=%s", s.conf.Bucket, s.conf.Domain)
	resp, err := s.s3get(addr)
	if err != nil {
		s.logger.Error().Err(err).Str("addr", addr).Msg("UpdateDomain")
		return
	}
	fmt.Println(resp)
}

func (s *Parser) UpdateDomain() {
	addr := fmt.Sprintf("http://%s.mls.cn-east-1.qiniumiku.com/?domainConfig&name=%s", s.conf.Bucket, s.conf.Domain)
	body := `{"enable": true}`
	resp, err := s.s3patch(addr, body)
	if err != nil {
		s.logger.Error().Err(err).Str("addr", addr).Msg("UpdateDomain")
		return
	}
	fmt.Println(resp)
}

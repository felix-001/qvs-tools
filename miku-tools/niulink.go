package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/qbox/mikud-live/common/auth/qiniumac.v1"
)

func (s *Parser) NiuLink() {
	addr := fmt.Sprintf("http://%s%s?page=%d&size=%d", s.conf.Domain, s.conf.NiulinkPath, 4, 1000)
	u, err := url.Parse(addr)
	if err != nil {
		s.logger.Error().Err(err).Msg("url.Parse")
		return
	}
	host := u.Host
	u.Host = ""
	u.Scheme = ""
	headers := map[string]string{}
	headers["Content-Type"] = "application/json"
	token := qiniumac.SignTokenWithParam(s.conf.Ak, s.conf.Sk, http.MethodGet, u.String(), host, "", headers)
	headers["Authorization"] = token
	resp, err := httpReq("GET", addr, "", headers)
	if err != nil {
		s.logger.Error().Err(err).Msg("http get")
		return
	}
	fmt.Println(resp)
}

package util

import (
	"fmt"
	"middle-source-analysis/config"
	"net/http"
	"net/url"

	"github.com/qbox/mikud-live/common/auth/qiniumac.v1"
)

func NiuLink(conf *config.Config) {
	addr := fmt.Sprintf("http://%s%s?page=%d&size=%d", conf.Domain, conf.NiulinkPath, 4, 1000)
	u, err := url.Parse(addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	host := u.Host
	u.Host = ""
	u.Scheme = ""
	headers := map[string]string{}
	headers["Content-Type"] = "application/json"
	token := qiniumac.SignTokenWithParam(conf.Ak, conf.Sk, http.MethodGet, u.String(), host, "", headers)
	headers["Authorization"] = token
	resp, err := HttpReq("GET", addr, "", headers)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp)
}

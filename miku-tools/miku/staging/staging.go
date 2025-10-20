package staging

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
)

type DnsResponse struct {
	Code      string `json:"code"`
	ConnectID string `json:"connectId"`
	Dns       []struct {
		Domain string   `json:"domain"`
		IPs    []string `json:"ips"`
		IPsV6  []string `json:"ipsv6"`
		TTL    int      `json:"ttl"`
	} `json:"dns"`
	Message string `json:"message"`
	UserIP  string `json:"userIp"`
	UserISP string `json:"userIsp"`
	UserLoc string `json:"userLoc"`
}

func TestDns(conf *config.Config, res *resources.Resources) {
	if conf.Uid != "" {
		conf.Ak, conf.Sk = util.GetAkSk(conf)
	} else {
		log.Println("need uid")
		return
	}
	if conf.Domain == "" {
		log.Println("need domain")
		return
	}
	errCnt := 0
	isps := map[string]bool{}
	errMap := map[string]int{}
	for isp, provIp := range res.V4Ips {
		for prov, ip := range provIp {
			addr := fmt.Sprintf("http://%s/?dns&domain=www.qiniu.com&ip=%s&type=0", conf.Domain, ip)
			respStr, err := util.QnHttpReq("GET", addr, "", conf.Ak, conf.Sk, map[string]string{})
			if err != nil {
			}
			resp := &DnsResponse{}
			if err := json.Unmarshal([]byte(respStr), resp); err != nil {
				log.Println("unmarshal err", err, resp, respStr)
				continue
			}
			if resp.Code != "1000" {
				log.Printf("dns test failed isp:%s prov:%s ip:%s code:%s message:%s\n",
					isp, prov, ip, resp.Code, resp.Message)
				errCnt++
				errMap[resp.Message]++
				continue
			}
			if len(resp.Dns) == 0 {
				log.Printf("dns test no dns records isp:%s prov:%s ip:%s\n", isp, prov, ip)
				errCnt++
				errMap["no dns records"]++
				continue
			}
			if len(resp.Dns[0].IPs) == 0 {
				errCnt++
				log.Printf("dns test no ipv4 records isp:%s prov:%s ip:%s\n", isp, prov, ip)
				errMap["no ipv4 records"]++
				continue
			}
			if len(resp.Dns[0].IPsV6) == 0 {
				errCnt++
				log.Printf("dns test no ipv6 records isp:%s prov:%s ip:%s, resp: %+v\n", isp, prov, ip, respStr)
				isps[isp] = true
				errMap["no ipv6 records"]++
				continue
			} else {
				/*
					log.Printf("dns test success isp:%s prov:%s ip:%s ipv4:%+v ipv6:%+v\n",
						isp, prov, ip, resp.Dns[0].IPs, resp.Dns[0].IPsV6)
				*/

			}
		}
	}
	log.Println("dns test done, err cnt:", errCnt)
	log.Printf("dns test lack ipv6 isps: %+v\n", isps)
	log.Printf("dns test err map: %+v\n", errMap)
}

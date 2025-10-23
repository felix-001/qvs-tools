package staging

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
	"net/http"
	"net/url"
	"strings"
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

const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func RandAlphaNum(n int) (string, error) {
	b := make([]byte, n)
	l := big.NewInt(int64(len(letters)))
	for i := range b {
		r, err := rand.Int(rand.Reader, l)
		if err != nil {
			return "", err
		}
		b[i] = letters[r.Int64()]
	}
	return string(b), nil
}

func TestIqiyi(conf *config.Config, res *resources.Resources) {
	for range 100 {
		streamId, err := RandAlphaNum(15)
		if err != nil {
			log.Println("rand alphanum err", err)
			return
		}
		area := get302(streamId, res)
		log.Println("iqiyi test done, streamId:", streamId, " area:", area)
	}
}

func get302(streamId string, res *resources.Resources) string {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 返回此错误会停止重定向，返回原始响应
			return http.ErrUseLastResponse
		},
	}
	// 发起 HTTP 请求
	resp, err := client.Get(fmt.Sprintf("http://flv-qnplay.inter.71edge.com/live/%s.flv", streamId))
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// 检查是否为 302 重定向
	if resp.StatusCode != http.StatusFound {
		fmt.Printf("未收到 302 响应，状态码: %d, resp: %+v\n", resp.StatusCode, resp)
		return ""
	}

	// 解析 Location 头部
	location := resp.Header.Get("Location")
	if location == "" {
		fmt.Println("未找到 Location 头部")
		return ""
	}

	// 提取 IP 地址
	parsedURL, err := url.Parse(location)
	if err != nil {
		fmt.Printf("解析 URL 失败: %v\n", err)
		return ""
	}

	host := parsedURL.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	fmt.Printf("提取的 IP 地址: %s, location: %s\n", host, location)
	_, area, _ := util.GetLocate(host, res.IpParser)
	//log.Println(isp, area, region)
	return area
}

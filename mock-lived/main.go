package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
)

type routeInfo struct {
	path    string
	handler http.HandlerFunc
}

type Rule struct {
	Pattern string `json:"pattern" bson:"pattern"`
	Replace string `json:"replace" bson:"replace"`
}

type StreamInfo struct {
	Key      string `json:"key"`
	Uid      int    `json:"uid"`
	Bucket   string `json:"bucket"`
	Disabled bool   `json:"disabled"`
}

type BucketInfo struct {
	BucketId     string   `json:"bucket"`
	User         string   `json:"user"` // 用户，通过Uid获得
	Uid          uint32   `json:"uid"`
	Status       string   `json:"status"`
	Redirect     int      `json:"redirect"`
	PiliAuth     int      `json:"piliAuth"`
	CallbackUrl  string   `json:"callbackUrl"`  // 通知、鉴权等回调
	AudioModel   int      `json:"audioModel"`   // 0:默认透传音频 1:如果是非aac,音频需要转码成aac 2:禁用音频 3:占位符，待拓展
	ForwardModel int      `json:"forwardModel"` // 0:rtmp 1:rtp 2:todo
	Domains      []string `json:"domains"`
	//StreamConf   *StreamConfig `json:"streamConf"`
}

type DomainInfo struct {
	Name     string `json:"name"`
	Bucket   string `json:"bucket"`
	User     string `json:"user"`
	Redirect int    `json:"redirect"` // <=0:默认关闭302, >0时开启302的比例，例如50时，代表50%比例
	PiliAuth int    `json:"piliAuth"` // <=0 关闭pili回源鉴权,  >0:开启pili鉴权
	//Auth          *AuthConfig    `json:"auth"`      // 防盗链配置参数信息
	//ThirdAuth     *UrlAuthConfig `json:"thirdAuth"` // 访问第三方url进行回源鉴权
	//StreamConf    *StreamConfig `json:"streamConf"`
	UrlRewrites   []Rule `json:"urlRewrites"`
	TsUrlRewrites []Rule `json:"tsUrlRewrites"`
}

type KeyStoreInfo struct {
	ErrCode string `json:"code"`
	Message string `json:"message"`
	//NodeInfo *RtNode `json:"node"`
	Action string `json:"action"`
	//Certs      []CertInfo      `json:"certs"`
	Domains []DomainInfo `json:"domains"`
	Streams []StreamInfo `json:"streams"`
	Buckets []BucketInfo `json:"buckets"`
	//GlobalCfgs []GlobalCfgInfo `json:"configs"`
	ConnectId string `json:"connectId"`
}

const (
	bucket1           = "bkt1"
	bucket2           = "bkt2"
	pullDomain        = "pull.qnlinking.com"
	pushDomain        = "push.qnlinking.com"
	pullDomain2       = "live.qnlinking.com"
	userNetease       = "netease"
	neteasePattern    = "(.+)/qiniu/(.+)$"
	neteaseReplace    = "${1}/live/${2}"
	neteaseTsPattern0 = "(.+)/playlist-(\\d{1,}).ts(\\?.*| *$)"
	neteaseTsReplace0 = "${1}/${2}.ts${3}"
	neteaseTsPattern1 = "(.+)/(.+)-(\\d{1,}).ts(\\?.*| *$)"
	neteaseTsReplace1 = "${1}/${2}/${3}.ts${4}"
)

func keyrequest(w http.ResponseWriter, req *http.Request) {
	data := &KeyStoreInfo{
		Domains: []DomainInfo{
			{
				Bucket: bucket1,
				Name:   pullDomain,
				User:   userNetease,
				UrlRewrites: []Rule{
					{
						Pattern: neteasePattern,
						Replace: neteaseReplace,
					},
				},
				TsUrlRewrites: []Rule{
					{
						Pattern: neteaseTsPattern0,
						Replace: neteaseTsReplace0,
					},
					{
						Pattern: neteaseTsPattern1,
						Replace: neteaseTsReplace1,
					},
				},
			},
			{
				Bucket: bucket1,
				Name:   pushDomain,
				User:   userNetease,
				UrlRewrites: []Rule{
					{
						Pattern: neteasePattern,
						Replace: neteaseReplace,
					},
				},
			},
			{
				Bucket: bucket2,
				Name:   pullDomain2,
				User:   userNetease,
			},
		},
		Buckets: []BucketInfo{
			{
				BucketId: bucket1,
			},
			{
				BucketId: bucket2,
			},
		},
	}
	resp, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintln(w, string(resp))
}

var routePath = []routeInfo{
	{path: "/api/v1/keyrequest", handler: keyrequest},
}

func route() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("url:", r.URL.Path)
		for _, route := range routePath {
			ok, err := regexp.Match(route.path, []byte(r.URL.Path))
			if err != nil {
				fmt.Println(err.Error())
			}
			if ok {
				route.handler(w, r)
				return
			}
		}
		if _, err := w.Write([]byte("404 not found")); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conn, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		log.Fatal(err)
	}
	if err := http.Serve(conn, route()); err != nil {
		log.Fatal(err)
	}
}

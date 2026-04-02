package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	qnconfig "github.com/qbox/bo-sdk/sdk/qconf/qconfapi/config"
	"github.com/qbox/mikud-live/cmd/dnspod/config"
	"github.com/qbox/mikud-live/common/dal"
	"github.com/qbox/pili/common/ipdb.v1"
	"gopkg.in/yaml.v3"
)

type CkConfig struct {
	Host   []string `json:"host" yaml:"host"`
	DB     string   `json:"db" yaml:"db"`
	User   string   `json:"user" yaml:"user"`
	Passwd string   `json:"passwd" yaml:"passwd"`
	Table  string   `json:"table" yaml:"table"`
}

type TCPRetranFilterConfig struct {
	Enable                 bool    `json:"enable" yaml:"enable"`                                       // 是否开启
	TCPRetranRateThreshold float64 `json:"tcp_retran_rate_threshold" yaml:"tcp_retran_rate_threshold"` // TCP 重传率阈值
	MaxFilterNodes         int     `json:"max_filter_nodes" yaml:"max_filter_nodes"`                   // TCP 重传率过滤最大节点数量
	MaxFilterIPs           int     `json:"max_filter_ips" yaml:"max_filter_ips"`                       // TCP 重传率过滤最大IP数量
	EnableUnicomFilter     bool    `json:"enable_unicom_filter" yaml:"enable_unicom_filter"`           // 是否过滤联通IP
	EnableTelecomFilter    bool    `json:"enable_telecom_filter" yaml:"enable_telecom_filter"`         // 是否过滤电信IP
	EnableMobileFilter     bool    `json:"enable_mobile_filter" yaml:"enable_mobile_filter"`           // 是否过滤移动IP
}

type IPSourceReqParam struct {
	AK              string `json:"ak" yaml:"ak"`
	SK              string `json:"sk" yaml:"sk"`
	LocalUrl        string `json:"local_url" yaml:"local_url"`
	RemoteUrl       string `json:"remote_url" yaml:"remote_url"`
	RemoteRetry     uint   `json:"retry_count" yaml:"retry_count"`
	RemoteExpireMS  uint   `json:"expire" yaml:"expire"`
	ReloadIntervalS int    `json:"reload_interval_s" yaml:"reload_interval_s"`
}

type IpdbConfig struct {
	IP map[string]*IPSourceReqParam `json:"ips_source_param" yaml:"ips_source_param"` //key: ipv4, ipv6
}

type ChartConf struct {
	Name            string    `json:"name" yaml:"name"`
	Title           string    `json:"title" yaml:"title"`
	SeriesTitle     string    `json:"seriesTitle" yaml:"seriesTitle"`
	Type            string    `json:"type" yaml:"type"`
	SQL             SQLConfig `json:"sql" yaml:"sql"`
	Table           string    `json:"table" yaml:"table"`
	RequireStreamId bool      `json:"requireStreamId" yaml:"requireStreamId"`
}

type SingleSQL struct {
	Select        string `json:"select" yaml:"select"`
	From          string `json:"from" yaml:"from"`
	Where         string `json:"where" yaml:"where"`
	GroupBy       string `json:"group_by" yaml:"group_by"`
	GroupByMinute bool   `json:"group_by_minute" yaml:"group_by_minute"`
	OrderBy       string `json:"order_by" yaml:"order_by"`
	Raw           string `json:"raw" yaml:"raw"`
	Table         string `json:"table" yaml:"table"`
}

type WithConfig struct {
	Name      string `json:"name" yaml:"name"`
	SingleSQL `json:"sql" yaml:"sql"`
}

type SQLConfig struct {
	With          []WithConfig `json:"with" yaml:"with"`
	Final         *SingleSQL   `json:"final" yaml:"final"`
	Fields        []string     `json:"fields" yaml:"fields"`
	PieName       string       `json:"pie_name" yaml:"pie_name"`
	PieValue      string       `json:"pie_value" yaml:"pie_value"`
	Dimension     string       `json:"dimension" yaml:"dimension"`
	GroupByMinute bool         `json:"group_by_minute" yaml:"group_by_minute"`
}

type Config struct {
	Cmd                   string
	Uid                   string
	Method                string
	Body                  string
	Addr                  string
	Help                  bool
	Https                 bool
	H                     bool
	Detail                bool
	Local                 bool
	Env                   string
	Output                string
	Pcdn                  string
	User                  string
	Passwd                string
	SchedIp               string
	Bucket                string
	Stream                string
	Domain                string
	Name                  string
	Key                   string
	Province              string
	SourceId              string
	OriginKey             string
	Origin                string
	Area                  string
	Isp                   string
	OriginKeyDy           string
	ID                    string
	OriginKeyHw           string
	Format                string
	Node                  string
	ConnId                string
	Ip                    string
	Query                 string
	App                   string
	RawApp                string
	Task                  string
	StartTime             string
	EndTime               string
	NsId                  string
	GBId                  string
	Host                  string
	QnTestUrl             string
	Player                string
	Bin                   string
	Process               int
	Basesub               int
	SubStream             int
	Startid               int
	Port                  int
	OnlineNum             int
	N                     int
	Loop                  int
	Bw                    int
	Offset                int
	Limit                 int
	F                     string
	T                     string
	Pattern               string
	Replace               string
	Raw                   string
	Protocol              string
	Skip                  string
	Url                   string
	JiraApiKeyFile        string
	ApiKey                string
	Redirect              bool
	Internal              bool
	Silence               bool
	Random                bool
	HeadReq               bool
	HeaderMap             HeaderMap
	CK                    CkConfig              `json:"ck" yaml:"ck"`
	Ak                    string                `json:"ak" yaml:"ak"`
	Sk                    string                `json:"sk" yaml:"sk"`
	Secret                string                `json:"secret" yaml:"secret"`
	IPDB                  ipdb.Config           `json:"ipdb" yaml:"ipdb_raw"`
	IpdbRaw               IpdbConfig            `json:"ipdb_raw" yaml:"ipdb"`
	RedisAddrs            []string              `json:"redis_addrs" yaml:"redis_addrs"`
	AccountCfg            qnconfig.Config       `json:"acc" yaml:"acc"`
	AccountCfgFile        string                `json:"acc_file" yaml:"acc_file"`
	DyApiSecret           string                `json:"dy_api_secret" yaml:"dy_api_secret"`
	DyApiDomain           string                `json:"dy_api_domain" yaml:"dy_api_domain"`
	MongoConf             *dal.MongoCfg         `json:"mongo_config" yaml:"mongo_config"`
	TcpRetranFilterConfig TCPRetranFilterConfig `json:"tcp_retran_filter_config" yaml:"tcp_retran_filter_config"`
	DnsPod                config.DnspodConfig   `json:"dnspod" yaml:"dnspod"`
	SMTPHost              string                `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort              int                   `json:"smtp_port" yaml:"smtp_port"`
	SMTPUser              string                `json:"smtp_user" yaml:"smtp_user"`
	SMTPPass              string                `json:"smtp_pass" yaml:"smtp_pass"`
	MailFrom              string                `json:"mail_from" yaml:"mail_from"`
	SMTPUseTLS            bool                  `json:"smtp_use_tls" yaml:"smtp_use_tls"`
	MailTo                string                `json:"mail_to" yaml:"mail_to"`
	Args                  []string              `json:"-" yaml:"-"` // 命令行参数，不进行JSON序列化
	ChartConfigs          []ChartConf           `json:"chart_configs" yaml:"chart_configs"`
	Path                  string                `json:"path" yaml:"path"`
}

func (c *Config) initIpdbConfig() {
	c.IPDB = ipdb.Config{IP: make(map[string]*ipdb.IPSourceReqParam)}
	for k, v := range c.IpdbRaw.IP {
		c.IPDB.IP[k] = &ipdb.IPSourceReqParam{
			AK:              v.AK,
			SK:              v.SK,
			LocalUrl:        v.LocalUrl,
			RemoteUrl:       v.RemoteUrl,
			RemoteRetry:     v.RemoteRetry,
			RemoteExpireMS:  v.RemoteExpireMS,
			ReloadIntervalS: time.Duration(v.ReloadIntervalS),
		}

	}
}

func Load() *Config {
	var conf Config
	conf.HeaderMap = make(map[string]string)
	conf.Args = os.Args[1:] // 保存命令行参数（排除程序名）
	_, err := os.Stat("/usr/local/etc/mikutool.yaml")
	if !os.IsNotExist(err) {
		data, err := os.ReadFile("/usr/local/etc/mikutool.yaml")
		if err != nil {
			log.Fatalf("读取文件失败: %v", err)
		}
		err = yaml.Unmarshal(data, &conf)
		//log.Printf("conf: %+v\n", conf)
		if err != nil {
			log.Fatalf("解析 YAML 失败: %v", err)
		}

		conf.initIpdbConfig()
		if conf.AccountCfgFile == "" {
			conf.AccountCfgFile = "/usr/local/etc/acc.json"
		}
		data, err = os.ReadFile(conf.AccountCfgFile)
		if err == nil {
			err = json.Unmarshal(data, &conf.AccountCfg)
			if err != nil {
				log.Printf("解析 json 失败: %v\n", err)
			} else {
				log.Printf("读取文件成功: %v\n", conf.AccountCfgFile)
			}
		} else {
			log.Printf("读取文件失败: %v, err: %v\n", conf.AccountCfgFile, err)
		}
		if conf.JiraApiKeyFile == "" {
			conf.JiraApiKeyFile = "/usr/local/etc/jira_api_key.conf"
		}
		data, err = os.ReadFile(conf.JiraApiKeyFile)
		if err != nil {
			log.Fatalf("读取文件失败: %v", err)
		}
		conf.ApiKey = string(data)[:len(data)-1]
		log.Printf("读取文件成功: %v, apikey; %s\n", conf.JiraApiKeyFile, conf.ApiKey)
		return &conf
	}
	_, err = os.Stat("/tmp/mikutool.yaml")
	if os.IsNotExist(err) {
		log.Fatalf("load config failed, err: %v", err)
		return nil
	}
	data, err := os.ReadFile("/tmp/mikutool.yaml")
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalf("解析 YAML 失败: %v", err)
	}

	conf.initIpdbConfig()
	log.Printf("conf: %+v\n", conf)
	return &conf
}
func (c *Config) ParseConsole() {
	flag.BoolVar(&c.Help, "help", false, "help")
	flag.StringVar(&c.Cmd, "cmd", "", "需要执行的命令")
	flag.StringVar(&c.Uid, "uid", "", "uid")
	flag.StringVar(&c.Method, "method", "", "method")
	flag.StringVar(&c.Body, "body", "", "body")
	flag.StringVar(&c.Addr, "addr", "", "addr")
	flag.StringVar(&c.Ak, "ak", "", "ak")
	flag.StringVar(&c.Sk, "sk", "", "sk")
	flag.StringVar(&c.Secret, "secret", "", "secret")
	flag.StringVar(&c.Pcdn, "pcdn", "", "pcdn")
	flag.StringVar(&c.Bucket, "bucket", "livessports", "bucket")
	flag.StringVar(&c.Stream, "stream", "teststream", "stream")
	flag.StringVar(&c.Domain, "domain", "www.test.com", "domain")
	flag.StringVar(&c.SourceId, "source_id", "", "source_id")
	flag.StringVar(&c.OriginKey, "origin_key", "", "origin_key")
	flag.StringVar(&c.OriginKeyDy, "origin_key_dy", "", "origin_key_dy")
	flag.StringVar(&c.OriginKeyHw, "origin_key_hw", "", "origin_key_hw")
	flag.StringVar(&c.Origin, "origin", "", "origin")
	flag.StringVar(&c.Area, "area", "华东", "area")
	flag.StringVar(&c.Isp, "isp", "电信", "isp")
	flag.StringVar(&c.F, "f", "", "f")
	flag.StringVar(&c.T, "t", "", "t")
	flag.StringVar(&c.Pattern, "pattern", "", "pattern")
	flag.StringVar(&c.Replace, "replace", "", "replace")
	flag.StringVar(&c.Raw, "raw", "", "raw")
	flag.StringVar(&c.User, "user", "iqiyi", "user")
	flag.StringVar(&c.Passwd, "passwd", "", "passwd")
	flag.StringVar(&c.SchedIp, "sched_ip", "10.34.146.62", "sched_ip")
	flag.StringVar(&c.Node, "node", "", "node")
	flag.StringVar(&c.ConnId, "conn_id", "12345678abcdef", "conn_id")
	flag.StringVar(&c.Ip, "ip", "103.85.174.230", "ip")
	flag.StringVar(&c.Query, "query", "", "query")
	flag.StringVar(&c.Format, "format", "flv", "format")
	flag.StringVar(&c.App, "app", "", "app")
	flag.StringVar(&c.Protocol, "protocol", "flv", "protocol")
	flag.StringVar(&c.Name, "name", "", "name")
	flag.StringVar(&c.Key, "key", "", "key")
	flag.StringVar(&c.Task, "task", "3995057", "task")
	flag.StringVar(&c.StartTime, "start", "", "start_time")
	flag.StringVar(&c.EndTime, "end", "", "end_time")
	flag.StringVar(&c.ID, "id", "", "id")
	flag.StringVar(&c.Province, "prov", "江苏", "province")
	flag.StringVar(&c.NsId, "nsid", "bj", "nsid")
	flag.StringVar(&c.GBId, "gbid", "31011500991320021895", "gbid")
	flag.StringVar(&c.Skip, "skip", "", "skip")
	flag.StringVar(&c.Host, "host", "", "host")
	flag.StringVar(&c.RawApp, "raw_app", "", "raw_app")
	flag.StringVar(&c.QnTestUrl, "qn_test_url", "", "qn_test_url")
	flag.StringVar(&c.Player, "player", "", "player")
	flag.StringVar(&c.Url, "url", "", "url")
	flag.StringVar(&c.MailTo, "mail_to", "", "mail_to")
	flag.StringVar(&c.Output, "output", "", "output")
	flag.IntVar(&c.Port, "port", 0, "port")
	flag.IntVar(&c.Basesub, "basesub", 0, "basesub")
	flag.IntVar(&c.SubStream, "substream", 0, "substream")
	flag.IntVar(&c.Startid, "startid", 0, "startid")
	flag.IntVar(&c.OnlineNum, "online_num", 10, "online_num")
	flag.IntVar(&c.Bw, "bw", 0, "bw")
	flag.BoolVar(&c.H, "h", false, "简版帮助信息, 如果需要详细的帮助信息, 请使用 -help")
	flag.BoolVar(&c.Https, "https", false, "是否https")
	flag.BoolVar(&c.Detail, "detail", false, "是否详细输出")
	flag.BoolVar(&c.Local, "local", false, "是否本地运行模式(不依赖redis/mongo等各种资源)")
	flag.BoolVar(&c.Redirect, "redirect", false, "是否开启302")
	flag.BoolVar(&c.Internal, "internal", false, "是否是内部的")
	flag.BoolVar(&c.Silence, "silence", false, "是否发送静音的语音数据")
	flag.BoolVar(&c.HeadReq, "head_req", false, "是否发送HEAD请求")
	flag.Var(&c.HeaderMap, "header", "header")
	flag.BoolVar(&c.Random, "random", false, "随机")
	flag.IntVar(&c.N, "n", 100, "n")
	flag.IntVar(&c.Loop, "loop", 10, "loop")
	flag.StringVar(&c.SMTPHost, "smtp_host", "", "SMTP服务器地址")
	flag.IntVar(&c.SMTPPort, "smtp_port", 587, "SMTP服务器端口")
	flag.StringVar(&c.SMTPUser, "smtp_user", "", "SMTP用户名")
	flag.StringVar(&c.SMTPPass, "smtp_pass", "", "SMTP密码")
	flag.StringVar(&c.MailFrom, "mail_from", "", "发件人邮箱")
	flag.IntVar(&c.Process, "process", 0, "进程id")
	flag.BoolVar(&c.SMTPUseTLS, "smtp_use_tls", true, "是否使用TLS")
	flag.StringVar(&c.Path, "path", "", "http api 请求的path")
	flag.StringVar(&c.AccountCfgFile, "accfile", "/usr/local/etc/acc.json", "账号配置文件")
	flag.IntVar(&c.Offset, "offset", 0, "offset")
	flag.IntVar(&c.Limit, "limit", 100, "limit")
	flag.StringVar(&c.JiraApiKeyFile, "api_key", "/usr/local/etc/jira_api_key.conf", "jira api key file")
	flag.StringVar(&c.Env, "env", "online", "env")
	flag.StringVar(&c.Bin, "bin", "sched", "bin")

	flag.Parse()
}

// HeaderMap 自定义类型用于存储多个 header
type HeaderMap map[string]string

// String 实现 flag.Value 接口的 String 方法
func (h *HeaderMap) String() string {
	return fmt.Sprintf("%v", *h)
}

// Set 实现 flag.Value 接口的 Set 方法
func (h *HeaderMap) Set(value string) error {
	if *h == nil {
		*h = make(HeaderMap)
	}

	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid header format: %s", value)
	}

	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])
	(*h)[key] = val
	return nil
}

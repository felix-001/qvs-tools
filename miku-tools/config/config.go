package config

import (
	"flag"
	"log"
	"os"

	qnconfig "github.com/qbox/bo-sdk/sdk/qconf/qconfapi/config"
	"github.com/qbox/mikud-live/cmd/dnspod/config"
	"github.com/qbox/mikud-live/common/dal"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
)

type CkConfig struct {
	Host   []string `json:"host"`
	DB     string   `json:"db"`
	User   string   `json:"user"`
	Passwd string   `json:"passwd"`
	Table  string   `json:"table"`
}

type TCPRetranFilterConfig struct {
	Enable                 bool    `json:"enable"`                    // 是否开启
	TCPRetranRateThreshold float64 `json:"tcp_retran_rate_threshold"` // TCP 重传率阈值
	MaxFilterNodes         int     `json:"max_filter_nodes"`          // TCP 重传率过滤最大节点数量
	MaxFilterIPs           int     `json:"max_filter_ips"`            // TCP 重传率过滤最大IP数量
	EnableUnicomFilter     bool    `json:"enable_unicom_filter"`      // 是否过滤联通IP
	EnableTelecomFilter    bool    `json:"enable_telecom_filter"`     // 是否过滤电信IP
	EnableMobileFilter     bool    `json:"enable_mobile_filter"`      // 是否过滤移动IP
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
	Task                  string
	StartTime             string
	EndTime               string
	Basesub               int
	SubStream             int
	Startid               int
	Port                  int
	OnlineNum             int
	F                     string
	T                     string
	Pattern               string
	Replace               string
	Raw                   string
	Protocol              string
	CK                    CkConfig              `json:"ck"`
	Ak                    string                `json:"ak"`
	Sk                    string                `json:"sk"`
	Secret                string                `json:"secret"`
	IPDB                  ipdb.Config           `json:"ipdb"`
	RedisAddrs            []string              `json:"redis_addrs"`
	AccountCfg            qnconfig.Config       `json:"acc"`
	DyApiSecret           string                `json:"dy_api_secret"`
	DyApiDomain           string                `json:"dy_api_domain"`
	MongoConf             *dal.MongoCfg         `json:"mongo_config"`
	TcpRetranFilterConfig TCPRetranFilterConfig `json:"tcp_retran_filter_config"`
	DnsPod                config.DnspodConfig   `json:"dnspod"`
}

func Load() *Config {
	var conf Config
	_, err := os.Stat("/usr/local/etc/mikutool.json")
	if !os.IsNotExist(err) {
		if err = qconfig.LoadFile(&conf, "/usr/local/etc/mikutool.json"); err != nil {
			log.Fatalf("load config failed, err: %v", err)
		}
		return &conf
	}
	_, err = os.Stat("/tmp/mikutool.json")
	if os.IsNotExist(err) {
		log.Fatalf("load config failed, err: %v", err)
		return nil
	}
	if err = qconfig.LoadFile(&conf, "/tmp/mikutool.json"); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
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
	flag.StringVar(&c.Domain, "domain", "flv-qnplay.inter.71edge.com", "domain")
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
	flag.StringVar(&c.Node, "node", "16234ef0-03f7-38f6-9bd7-003d0ba2081e-vdn-jsyz1-dls-1-9", "node")
	flag.StringVar(&c.ConnId, "conn_id", "12345678abcdef", "conn_id")
	flag.StringVar(&c.Ip, "ip", "111.51.164.99", "ip")
	flag.StringVar(&c.Query, "query", "", "query")
	flag.StringVar(&c.Format, "format", "flv", "format")
	flag.StringVar(&c.App, "app", "live", "app")
	flag.StringVar(&c.Protocol, "protocol", "flv", "protocol")
	flag.StringVar(&c.Name, "name", "", "name")
	flag.StringVar(&c.Key, "key", "", "key")
	flag.StringVar(&c.Task, "task", "3995057", "task")
	flag.StringVar(&c.StartTime, "start", "", "start_time")
	flag.StringVar(&c.EndTime, "end", "", "end_time")
	flag.StringVar(&c.ID, "id", "", "id")
	flag.IntVar(&c.Basesub, "basesub", 0, "basesub")
	flag.IntVar(&c.SubStream, "substream", 0, "substream")
	flag.IntVar(&c.Startid, "startid", 0, "startid")
	flag.IntVar(&c.OnlineNum, "online_num", 10, "online_num")
	flag.BoolVar(&c.H, "h", false, "简版帮助信息, 如果需要详细的帮助信息, 请使用 -help")
	flag.BoolVar(&c.Https, "https", false, "是否https")
	flag.BoolVar(&c.Detail, "detail", false, "是否详细输出")
	flag.BoolVar(&c.Local, "local", false, "是否本地运行模式(不依赖redis/mongo等各种资源)")

	flag.Parse()
}

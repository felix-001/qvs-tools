package config

import (
	"flag"
	"log"
	"middle-source-analysis/public"
	"os"
	"time"

	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"
	"github.com/qbox/mikud-live/cmd/dnspod/config"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
)

const (
	confFile = "/usr/local/etc/middle-source.json"
)

type CkConfig struct {
	Host   []string `json:"host"`
	DB     string   `json:"db"`
	User   string   `json:"user"`
	Passwd string   `json:"passwd"`
	Table  string   `json:"table"`
}

type Config struct {
	RedisAddrs     []string                      `json:"redis_addrs"`
	IPDB           ipdb.Config                   `json:"ipdb"`
	CK             CkConfig                      `json:"ck"`
	Secret         string                        `json:"secret"`
	PrometheusAddr string                        `json:"prometheus"`
	DyApiSecret    string                        `json:"dy_api_secret"`
	DyApiDomain    string                        `json:"dy_api_domain"`
	AccountCfg     qconfapi.Config               `json:"acc"`
	OriginKey      string                        `json:"origin_key"`
	OriginKeyDy    string                        `json:"origin_key_dy"`
	OriginKeyHw    string                        `json:"origin_key_hw"`
	NiulinkPath    string                        `json:"niulink_path"`
	KubeCfg        string                        `json:"kube_cfg"`
	DnsPod         config.DnspodConfig           `json:"dnspod"`
	SendKey        string                        `json:"send_key"`
	BwRatioConfig  map[string]map[string]float64 `json:"bw_ratio_config"` // key1: 大区 key2: isp(取值 移动/电信/联通/total)
	IdcBwConfig    map[string]map[string]int     `json:"idc_bw_config"`   // key1: idc key2: isp
	//Kodo               qvs.KODOConfig                `json:"kodo`
	Bucket             string
	SubCmd             string
	Stream             string
	Format             string
	Node               string
	NeedIpParer        bool
	NeedCk             bool
	NeedNodeStreamInfo bool
	LagFile            string
	NodeInfo           bool
	RootNodeInfo       bool
	Prometheus         bool
	Redis              bool
	DnsResFile         string
	PathqueryLogFile   string
	Ak                 string `json:"ak"`
	Sk                 string `json:"sk"`
	AdminAk            string `json:"admin_ak"`
	AdminSk            string `json:"admin_sk"`
	Cmd                string
	QosFile            string
	Help               bool
	Isp                string
	Province           string
	Area               string
	Pcdn               string
	Ip                 string
	T                  string
	Query              string
	N                  int
	QpmFile            string
	PcdnErr            string
	User               string
	ConnId             string
	Https              bool
	Interval           int
	SchedIp            string
	Uid                string
	Domain             string
	SourceId           string
	Origin             string
	Basesub            int
	SubStream          int
	Startid            int
	F                  string
	Force              bool
	RecordId           string
	Name               string
	Cnt                int
	TotoalNeedBw       int
	Method             string
	Body               string
	Addr               string
	Port               int
	ID                 string
	Transport          string
	Passwd             string
	Key                string
	Ns                 string
}

func LoadCfg() *Config {
	var conf Config
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		if _, err := os.Stat("/tmp/mikutool.json"); os.IsNotExist(err) {
			log.Fatalf("load config failed, err: %v", err)
		} else {
			if err = qconfig.LoadFile(&conf, "/tmp/mikutool.json"); err != nil {
				log.Fatalf("load config failed, err: %v", err)
			}
		}
	} else {
		if err := qconfig.LoadFile(&conf, confFile); err != nil {
			log.Fatalf("load config failed, err: %v", err)
		}
	}

	t := time.Now().Format("2006-01-02 15:04:05")

	flag.StringVar(&conf.Bucket, "bkt", "douyu", "bucket ID")
	flag.StringVar(&conf.Node, "node", "2b8f0c5a-85d0-3c4a-bbd8-ac77a82d607b-rtc-gdfsh-dls-1-7", "node ID")
	flag.StringVar(&conf.Stream, "stream", "288016rlols5_2000p", "stream ID")
	flag.BoolVar(&conf.NeedNodeStreamInfo, "streamNodes", false, "是否需要流所在的节点信息")
	flag.StringVar(&conf.LagFile, "lagfile", "", "分析streamd上报的卡顿数据的文件")
	flag.BoolVar(&conf.Prometheus, "prometheusEnable", false, "是否需要加载prometheus")
	flag.StringVar(&conf.DnsResFile, "dnschk", "", "阿里网络拨测工具结果文件")
	flag.StringVar(&conf.PathqueryLogFile, "pathqueryfile", "", "解析elk下载的pathquery日志文件,判断回源路径是否符合预期")
	flag.StringVar(&conf.Ak, "ak", conf.Ak, "ak")
	flag.StringVar(&conf.Sk, "sk", conf.Sk, "sk")
	flag.StringVar(&conf.Cmd, "cmd", "streams", "需要执行的子命令(hlschk)")
	flag.StringVar(&conf.QosFile, "qosfile", "", "解析从ck上下载的streamd qos文件，检查locate addr和remote addr是否跨isp、省、大区")
	flag.BoolVar(&conf.Help, "h", false, "help")
	flag.StringVar(&conf.Isp, "isp", "电信", "isp")
	flag.StringVar(&conf.Province, "province", "浙江", "省份")
	flag.StringVar(&conf.Area, "area", "华东", "大区")
	flag.StringVar(&conf.Pcdn, "pcdn", "", "指定pcdn的ip:port")
	flag.StringVar(&conf.Ip, "ip", "221.227.254.220", "通过ip获取node id")
	flag.StringVar(&conf.T, "t", t, "时间, 格式: 2006-01-02 15:04:05")
	flag.StringVar(&conf.Query, "q", "", "查询ck的语句")
	flag.IntVar(&conf.N, "n", 70, "pcdn循环请求次数")
	flag.StringVar(&conf.QpmFile, "qpmfile", "", "dump出来的内存中的qpm数据")
	flag.StringVar(&conf.PcdnErr, "pcdnerr", "", "请求pcdn是附带的pcdn err节点id")
	flag.StringVar(&conf.User, "user", "volcengine", "请求playcheck 302的user")
	flag.StringVar(&conf.ConnId, "connid", "testConnId", "请求playcheck 302的connId")
	flag.BoolVar(&conf.Https, "https", false, "是否启用https")
	flag.IntVar(&conf.Interval, "interval", 15, "时间间隔")
	flag.StringVar(&conf.SchedIp, "schedip", "10.34.146.62", "调度服务ip")
	flag.StringVar(&conf.Uid, "uid", "", "uid")
	flag.StringVar(&conf.Domain, "domain", "www.voltest2.com", "domain")
	flag.StringVar(&conf.SubCmd, "subcmd", "", "staging subcmd")
	flag.StringVar(&conf.Format, "format", "slice", "playcheck请求原始url的format, 例如: flv m3u8 xs slice")
	flag.StringVar(&conf.SourceId, "sourceid", "2275133575", "播放dy流的sourceid")
	flag.StringVar(&conf.Origin, "origin", "", "tct/dy/hw")
	flag.IntVar(&conf.Basesub, "basesub", 1, "子流总数，切片ID按此取模得到切片所属子流号")
	flag.IntVar(&conf.SubStream, "substream", 0, "子流号(以0开始，比如例子中 0,1,2,3,4,5)")
	flag.IntVar(&conf.Startid, "startid", 0, "起始发送切片ID")
	flag.StringVar(&conf.F, "f", "/tmp/out.xs", "输出文件")
	flag.BoolVar(&conf.Force, "force", false, "是否强制刷新")
	flag.StringVar(&conf.RecordId, "recordid", "", "dns record id, 支持,分割多条记录")
	flag.StringVar(&conf.Name, "name", "extension", "dns name")
	flag.IntVar(&conf.Cnt, "cnt", 1, "count")
	flag.IntVar(&conf.TotoalNeedBw, "needbw", 100, "需要的建设带宽，单位为G")
	flag.StringVar(&conf.Method, "method", "", "http请求的方法")
	flag.StringVar(&conf.Body, "body", "", "http请求的body")
	flag.StringVar(&conf.Addr, "addr", "", "http请求的url")
	flag.IntVar(&conf.Port, "port", 7279, "http请求的端口")
	flag.StringVar(&conf.ID, "id", "31011500991180000130", "gbid")
	flag.StringVar(&conf.Transport, "s", "udp", "Transport")
	flag.StringVar(&conf.Passwd, "p", "123456", "Password")
	flag.StringVar(&conf.Key, "key", "", "kodo key")
	flag.StringVar(&conf.Ns, "ns", "FengTaiWiFi", "namespace id")

	flag.Parse()

	if conf.Cmd == "ispchk" {
		conf.Redis = true
		conf.NeedIpParer = true
		conf.NodeInfo = true
	}

	if conf.Cmd == "hlschk" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedNodeStreamInfo = true
	}
	if conf.Cmd == "streams" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedNodeStreamInfo = true
		conf.NeedIpParer = true
	}
	if conf.Cmd == "bwdis" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedNodeStreamInfo = true
		conf.NeedIpParer = true
	}
	if conf.Cmd == "coverchk" {
		conf.NeedIpParer = true
	}
	if conf.Cmd == "dumproot" {
		conf.Redis = true
		conf.NodeInfo = true
	}
	if conf.Cmd == "nodebyip" {
		conf.Redis = true
		conf.NodeInfo = true
	}
	if conf.Cmd == "pcdns" {
		conf.NeedIpParer = true
	}
	if conf.Cmd == "nali" {
		conf.NeedIpParer = true
	}
	if conf.Cmd == "dytimeout" {
		conf.Redis = true
		conf.NodeInfo = true
	}
	if conf.Cmd == "stag" && conf.SubCmd == "ipv6" {
		conf.Redis = true
		conf.NodeInfo = true
		conf.NeedIpParer = true
	}
	return &conf
}

func InitConf(cmdMap map[string]public.CmdInfo, cfg *Config) {
	for cmd, cmdInfo := range cmdMap {
		if cfg.Cmd != cmd {
			continue
		}
		for _, conf := range cmdInfo.Depends {
			log.Println(cmd, "initConf")
			*conf = true
		}
	}
}

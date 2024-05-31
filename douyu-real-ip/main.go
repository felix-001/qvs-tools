package main

import (
	"flag"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
)

var (
	configFile = flag.String("f", "/usr/local/etc/miku.json", "the config file")
)

type Config struct {
	IPDB         ipdb.Config `json:"ipdb"`
	ClientIpFile string      `json:"client_ip_file"`
	PcdnIpFile   string      `json:"pcdn_ip_file"`
}

type Parser struct {
	ipparser *ipdb.City
	conf     *Config
}

func (s *Parser) getIp(input string) string {
	//ipRegex := `(?m)\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`
	//re := regexp.MustCompile(ipRegex)
	//ip := re.FindString(input)
	start := strings.Index(input, "[")
	start += 2
	input = input[start:]
	end := strings.Index(input, ",")
	return input[:end]
}

func (s *Parser) ClientIpListAyalyze() map[string]int {
	b, err := ioutil.ReadFile(s.conf.ClientIpFile)
	if err != nil {
		log.Fatalln("read fail", s.conf, err)
	}
	lines := strings.Split(string(b), "\n")
	log.Println(len(lines))
	info := map[string]int{}
	for _, line := range lines {
		ip := s.getIp(line)
		//log.Println(ip)
		info[ip]++
	}
	log.Println(len(info))
	return info
}

func (s *Parser) getClientIp(line string) string {
	start := strings.Index(line, "clientIp=")
	if start == -1 {
		//log.Println("find start err", line)
		return ""
	}
	start += len("clientIp=")
	line = line[start:]
	end := strings.Index(line, "&pcdn_error")
	if end == -1 {
		//log.Println("find end err", line)
		return ""
	}
	return line[:end]
}

func (s *Parser) getStream(line string) string {
	start := strings.Index(line, "req:")
	if start == -1 {
		return ""
	}
	start += len("req:")
	line = line[start:]
	ss := strings.Split(line, "/")
	if len(ss) != 4 {
		log.Println("len(ss)", len(ss))
		return ""
	}
	//log.Println(ss[2])
	return ss[2]
}

func sortMap(m map[string]int) []Pair {
	pairs := []Pair{}
	for k, v := range m {
		pairs = append(pairs, Pair{Key: k, Val: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Val > pairs[j].Val
	})
	return pairs
}

type Pair struct {
	Key string
	Val int
}

func (s *Parser) PcdnReqAnalyze() map[string]int {
	b, err := ioutil.ReadFile(s.conf.PcdnIpFile)
	if err != nil {
		log.Fatalln("read fail", s.conf, err)
	}
	lines := strings.Split(string(b), "\n")
	log.Println("PcdnReqAnalyze", len(lines))
	info := map[string]int{}
	streams := map[string]int{}
	areaInfo := map[string]int{}
	for _, line := range lines {
		ip := s.getClientIp(line)
		if ip == "" {
			continue
		}
		locate, err := s.ipparser.Find(ip)
		if err != nil {
			log.Println("get locate of ip", ip, "err", err)
			continue
		}
		areaIpsKey, _ := util.GetAreaIspKey(locate)
		if !strings.Contains(areaIpsKey, "华东") {
			continue
		}
		areaInfo[areaIpsKey]++
		//log.Println(ip)
		info[ip]++
		stream := s.getStream(line)
		if stream == "" {
			continue
		}
		streams[stream]++
	}
	log.Println(streams)
	log.Println("PcdnReqAnalyze ips", len(info), "streams", len(streams))
	log.Println(areaInfo)
	ips := sortMap(info)
	for _, ip := range ips {
		log.Println(ip.Key, ip.Val)
	}
	return info
}

func (s *Parser) Merge() {
	blackIps := s.ClientIpListAyalyze()
	ips := s.PcdnReqAnalyze()
	iplist := []string{}
	for ip := range ips {
		if _, ok := blackIps[ip]; !ok {
			iplist = append(iplist, ip)
		}
	}
	//log.Println(iplist, len(iplist))
	log.Println("iplist:", len(iplist))
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conf := &Config{}
	err := qconfig.LoadFile(conf, *configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s\n", err.Error())
	}
	ipParser, err := ipdb.NewCity(conf.IPDB)
	if err != nil {
		log.Fatalf("[IPDB NewCity] err: %+v\n", err)
	}
	parser := Parser{ipparser: ipParser, conf: conf}
	//parser.ClientIpListAyalyze()
	//parser.PcdnReqAnalyze()
	parser.Merge()
}

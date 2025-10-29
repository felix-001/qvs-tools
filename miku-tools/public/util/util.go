package util

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qbox/bo-sdk/base/xlog.v1"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"
	schedUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
	schedModel "github.com/qbox/mikud-live/cmd/sched/model"
	"github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/rs/zerolog"
)

func Str2unix(s string) (int64, error) {
	loc, _ := time.LoadLocation("Local")
	the_time, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return the_time.Unix(), nil
}

func Str2time(s string) (time.Time, error) {
	loc, _ := time.LoadLocation("Local")
	return time.ParseInLocation("2006-01-02 15:04:05", s, loc)
}

type Pair struct {
	Key   string
	Value int
}

func SortIntMap(m map[string]int) []Pair {
	pairs := make([]Pair, 0)
	for k, v := range m {
		pairs = append(pairs, Pair{Key: k, Value: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})
	return pairs
}

func DumpSlice(pairs []Pair) {
	for _, pair := range pairs {
		fmt.Println(pair.Key, pair.Value)
	}
}

func IsIpv6(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		fmt.Println("IP address is not valid")
		return false
	}
	if ip.To4() == nil {
		return true
	} else {
		return false
	}
}

func ipv4Net(a, b, c, d byte, subnetPrefixLen int) net.IPNet {
	return net.IPNet{IP: net.IPv4(a, b, c, d), Mask: net.CIDRMask(96+subnetPrefixLen, 128)}
}

var reservedIPv4Nets = []net.IPNet{
	ipv4Net(0, 0, 0, 0, 8),       // Current network
	ipv4Net(10, 0, 0, 0, 8),      // Private
	ipv4Net(100, 64, 0, 0, 10),   // RFC6598
	ipv4Net(127, 0, 0, 0, 8),     // Loopback
	ipv4Net(169, 254, 0, 0, 16),  // Link-local
	ipv4Net(172, 16, 0, 0, 12),   // Private
	ipv4Net(192, 0, 0, 0, 24),    // RFC6890
	ipv4Net(192, 0, 2, 0, 24),    // Test, doc, examples
	ipv4Net(192, 88, 99, 0, 24),  // IPv6 to IPv4 relay
	ipv4Net(192, 168, 0, 0, 16),  // Private
	ipv4Net(198, 18, 0, 0, 15),   // Benchmarking tests
	ipv4Net(198, 51, 100, 0, 24), // Test, doc, examples
	ipv4Net(203, 0, 113, 0, 24),  // Test, doc, examples
	ipv4Net(224, 0, 0, 0, 4),     // Multicast
	ipv4Net(240, 0, 0, 0, 4),     // Reserved (includes broadcast / 255.255.255.255)
}
var globalUnicastIPv6Net = net.IPNet{IP: net.IP{0x20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: net.CIDRMask(3, 128)}

func isIPv6GlobalUnicast(address net.IP) bool {
	return globalUnicastIPv6Net.Contains(address)
}

func isIPv4Reserved(address net.IP) bool {
	for _, reservedNet := range reservedIPv4Nets {
		if reservedNet.Contains(address) {
			return true
		}
	}
	return false
}

func isPublicIPAddress(address net.IP) bool {
	return isIPv6GlobalUnicast(address) || (address.To4() != nil && !isIPv4Reserved(address))
}

func IsPublicIPAddress(ip string) bool {
	return isPublicIPAddress(net.ParseIP(ip))
}

func GetAkSk(conf *config.Config) (string, string) {
	qc := qconfapi.New(&conf.AccountCfg)
	ag := appg.Client{Conn: qc}
	uid, err := strconv.Atoi(conf.Uid)
	if err != nil {
		log.Fatalln(err)
	}
	ak, sk, err := ag.GetAkSk(xlog.FromContextSafe(context.Background()), uint32(uid))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("ak:", ak, "sk:", sk)
	return ak, sk
}

func Province2Area(conf *config.Config) {
	parts := strings.Split(conf.Province, ",")

	result := ""
	for _, province := range parts {
		area, _ := schedUtil.ProvinceAreaRelation(province)
		result += area + ","
	}
	log.Println(result)
}

func GetLocate(ip string, ipParser *ipdb.City) (string, string, string) {
	locate, err := ipParser.Find(ip)
	if err != nil {
		log.Println(err, ip)
		return "", "", ""
	}
	if locate.Isp == "" {
		//log.Println("country", locate.Country, "isp", locate.Isp, "city", locate.City, "region", locate.Region, "ip", ip)
	}
	if locate.Country != "中国" {
		log.Println("country", locate.Country, "isp", locate.Isp, "city", locate.City, "region", locate.Region, "ip", ip)
	}
	area, _ := schedUtil.ProvinceAreaRelation(locate.Region)
	return locate.Isp, area, locate.Region
}

func UploadFile(filePath string) {
	cmd := exec.Command("qup", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

// key1: isp key2: 省份
func LoadV4Ips() (ipMap map[string]map[string]string) {
	ipMap = make(map[string]map[string]string)
	if err := json.Unmarshal([]byte(RawIpv4s), &ipMap); err != nil {
		log.Println(err)
		return
	}
	return
}

func LoadV6Ips() (ipMap map[string]map[string]string) {
	ipMap = make(map[string]map[string]string)
	lines := strings.Split(RawIpv6s, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			continue
		}
		region := strings.TrimSpace(fields[0])
		isp := strings.TrimSpace(fields[1])
		remoteAddr := strings.TrimSpace(fields[4])
		ip, _, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			// 如果解析失败，跳过当前行
			continue
		}
		if _, ok := ipMap[isp]; !ok {
			ipMap[isp] = make(map[string]string)
		}
		ipMap[isp][region] = ip
	}
	return
}

var sublogger = zerolog.New(os.Stdout).With().Timestamp().Logger()

func GetPcdnFromSchedAPI(conf *config.Config) (string, string) {
	addr := "http://10.34.146.62:6060/api/v1/nodes?level=default&dimension=area&mode=detail&ipversion=ipv4"
	resp, err := Get(addr)
	if err != nil {
		sublogger.Error().Err(err).Str("addr", addr).Msg("get nodes err")
		return "", ""
	}
	//fmt.Println(resp)
	areaNodesMap := make(map[string][]*schedModel.NodeIpsPair)
	if err := json.Unmarshal([]byte(resp), &areaNodesMap); err != nil {
		sublogger.Error().Err(err).Msg("unmashal err")
		return "", ""
	}
	key := fmt.Sprintf("area_isp_group_%s_%s", conf.Area, conf.Isp)
	nodes, ok := areaNodesMap[key]
	if !ok {
		sublogger.Error().
			Str("area", conf.Area).
			Str("isp", conf.Isp).
			Msg("area isp not found nodes")
		return "", ""
	}
	if len(nodes) == 0 {
		sublogger.Error().Msg("nodes len is 0")
		return "", ""
	}
	pcdn := ""
	var selectNode *schedModel.NodeIpsPair
	for _, nodeInfo := range nodes {
		for _, ipInfo := range nodeInfo.Ips {
			if ipInfo.IsIPv6 {
				continue
			}
			if util.IsPrivateIP(ipInfo.Ip) {
				continue
			}
			pcdn = fmt.Sprintf("%s:%d", ipInfo.Ip, nodeInfo.Node.StreamdPorts.Http)
			selectNode = nodeInfo
			break
		}
	}
	if pcdn == "" {
		sublogger.Error().Msg("pcdn empty")
		return "", ""
	}
	sublogger.Info().Str("nodeId", selectNode.Node.Id).Str("machineId", selectNode.Node.MachineId).Msg("selected node")
	return selectNode.Node.Id, pcdn
}

// InetAton converts an IPv4 string to a uint32, similar to inet_aton in C.
func InetAton(ipStr string) (uint32, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP: %s", ipStr)
	}
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("not an IPv4 address: %s", ipStr)
	}
	return uint32(ip[0]) | uint32(ip[1])<<8 | uint32(ip[2])<<16 | uint32(ip[3])<<24, nil
}

package util

import (
	"context"
	"fmt"
	"log"
	"mikutool/config"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qbox/bo-sdk/base/xlog.v1"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"
	schedUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
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

func GetAkSk(conf *config.Config) {
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

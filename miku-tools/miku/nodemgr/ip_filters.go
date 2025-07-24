package nodemgr

import (
	"log"
	"mikutool/config"
	"mikutool/resources"

	"github.com/qbox/mikud-live/cmd/sched/dal"
	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/rs/zerolog"
)

type IpFilter interface {
	Filter(node *public.RtNode, ip *public.RtIpStatus) bool
	Name() string
	Enable() bool
	SetEnable(enable bool)
}
type LossFilter struct {
	Switch
	lossIpsMap map[string]bool
	resources  resources.Resources
}

func (f *LossFilter) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	key := GetIpPingKey(node.Id, ip.Ip)
	return !f.lossIpsMap[key]
}

func (f *LossFilter) Name() string {
	return "Loss"
}

func (f *LossFilter) EventHandler(data any) {
	resources, ok := data.(resources.Resources)
	if !ok {
		return
	}
	f.resources = resources
	if f.resources.Redis == nil {
		return
	}
	lossMap, err := dal.GetPingLossIps(zerolog.Logger{}, resources.Redis)
	if err != nil {
		log.Println("LossFilter EventHandler GetPingLossIps err")
		return
	}
	f.lossIpsMap = lossMap
}

type RttFilter struct {
	Switch
	resources resources.Resources
	rttIpsMap map[string]bool
}

func (f *RttFilter) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	key := GetIpPingKey(node.Id, ip.Ip)
	return !f.rttIpsMap[key]
}

func (f *RttFilter) Name() string {
	return "Rtt"
}

func (f *RttFilter) EventHandler(data any) {
	resources, ok := data.(resources.Resources)
	if !ok {
		return
	}
	f.resources = resources
	if f.resources.Redis == nil {
		return
	}
	rttMap, err := dal.GetPingRttIps(zerolog.Logger{}, resources.Redis)
	if err != nil {
		log.Println("RttFilter EventHandler GetPingRttIps err")
		return
	}
	f.rttIpsMap = rttMap
}

type IpFilterPrivate struct {
	Switch
}

func (f *IpFilterPrivate) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	return !publicUtil.IsPrivateIP(ip.Ip)
}

func (f *IpFilterPrivate) Name() string {
	return "PrivateIp"
}

type IpFilterForbidden struct {
	Switch
}

func (f *IpFilterForbidden) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	return !ip.Forbidden
}

func (f *IpFilterForbidden) Name() string {
	return "IpFrobidden"
}

type IpFilterProbeSpeed struct {
	Switch
}

func (f *IpFilterProbeSpeed) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	if ip.IPStreamProbe.Speed > 0 && ip.IPStreamProbe.MinSpeed > 0 &&
		ip.IPStreamProbe.Speed < 8 &&
		ip.IPStreamProbe.MinSpeed < 6 {
		return false
	}
	return true
}

func (f *IpFilterProbeSpeed) Name() string {
	return "ProbeSpeed"
}

type IpFilterIpv6 struct {
	Switch
}

func (f *IpFilterIpv6) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	return publicUtil.IsIPv6(ip.Ip)
}

func (f *IpFilterIpv6) Name() string {
	return "Ipv6"
}

type IpFilterTcpRetrans struct {
	Switch
	tcpRetransMap map[string]float64
	conf          *config.Config
}

func (f *IpFilterTcpRetrans) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	if f.tcpRetransMap[ip.Ip] > f.conf.TcpRetranFilterConfig.TCPRetranRateThreshold {
		log.Println("IP filtered due to high TCP retransmission rate",
			"nodeId", node.Id,
			"machineId", node.MachineId,
			"ip", ip.Ip,
			"isp", ip.IpIsp,
			"rate", f.tcpRetransMap[ip.Ip])
		return false
	}
	return true
}

func (f *IpFilterTcpRetrans) Name() string {
	return "TcpRetrans"
}

func (f *IpFilterTcpRetrans) SetData(tcpRetransMap map[string]float64) {
	f.tcpRetransMap = tcpRetransMap
}

func (f *IpFilterTcpRetrans) SetConf(conf *config.Config) {
	f.conf = conf
}

type IpFilterAvailableBw struct {
	Switch
}

func (f *IpFilterAvailableBw) Filter(node *public.RtNode, ip *public.RtIpStatus) bool {
	return ip.OutMBps > ip.MaxOutMBps*0.93
}

func (f *IpFilterAvailableBw) Name() string {
	return "AvailableBw"
}

func GetIpPingKey(nodeId, ip string) string {
	return "ping_result_" + nodeId + "_" + ip
}

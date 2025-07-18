package nodemgr

import (
	"context"
	"log"
	"mikutool/resources"

	commonUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
)

type NodeFilter interface {
	Filter(node *public.RtNode) bool
	Name() string
	Enable() bool
}

type IpFilter interface {
	Filter(ip *public.RtIpStatus) bool
	Name() string
	Enable() bool
}

const (
	FilterTypeLocal = "local"
	FilterTypeMongo = "mongo"
)

type FilterMgr struct {
	nodeFilters      []NodeFilter
	ipFilters        []IpFilter
	statistics       map[string]int
	resources        resources.Resources
	nodeAvailability map[string]*commonUtil.NodeAvailabilityInfo
	filterType       string
}

type DynamicFilter struct {
	Switch
}

func (f *DynamicFilter) Filter(node *public.RtNode) bool {
	return node.IsDynamic
}

func (f *DynamicFilter) Name() string {
	return "Dynamic"
}

type NotBanProvFilter struct {
	Switch
}

func (f *NotBanProvFilter) Filter(node *public.RtNode) bool {
	return !node.IsBanTransProv
}

func (f *NotBanProvFilter) Name() string {
	return "NotBanProv"
}

type ServingFilter struct {
	Switch
}

func (f *ServingFilter) Filter(node *public.RtNode) bool {
	return node.RuntimeStatus == "Serving"
}

func (f *ServingFilter) Name() string {
	return "Serving"
}

func NodeFilterServing(node *public.RtNode) (string, bool) {
	return "NotServing", node.RuntimeStatus == "Serving"
}

type NoStreamdPortsFilter struct {
	Switch
}

func (f *NoStreamdPortsFilter) Filter(node *public.RtNode) bool {
	if node.StreamdPorts.Http == 0 {
		return false
	}
	if node.StreamdPorts.Wt == 0 {
		return false
	}
	if node.StreamdPorts.Https == 0 {
		return false
	}
	return true
}

func (f *NoStreamdPortsFilter) Name() string {
	return "NoStreamdPorts"
}

type AbilitiesFilter struct {
	Switch
}

func (f *AbilitiesFilter) Filter(node *public.RtNode) bool {
	ability, ok := node.Abilities["live"]
	if !ok || !ability.Can || ability.Frozen {
		return false
	}
	return true
}

func (f *AbilitiesFilter) Name() string {
	return "Abilities"
}

type ServicesFilter struct {
	Switch
}

func (f *ServicesFilter) Filter(node *public.RtNode) bool {
	if _, ok := node.Services["live"]; !ok {
		return false
	}
	return true
}

func (f *ServicesFilter) Name() string {
	return "Services"
}

type TimeLimitFilter struct {
	Switch
}

func (f *TimeLimitFilter) Filter(node *public.RtNode) bool {
	return len(node.Schedules) == 0
}

func (f *TimeLimitFilter) Name() string {
	return "TimeLimit"
}

type LossFilter struct {
	Switch
	lossIpsMap map[string]bool
}

func (f *LossFilter) Filter(ip *public.RtIpStatus) bool {
	// key := publicDal.GetIpPingKey(node.Id, ipIsp.Ip)
	//		if lossIpsMap[key] {
	// TODO
	return !f.lossIpsMap[ip.Ip]
}

func (f *LossFilter) Name() string {
	return "Loss"
}

type RttFilter struct {
	Switch
	rttIpsMap map[string]bool
}

func (f *RttFilter) Filter(ip *public.RtIpStatus) bool {
	// key := publicDal.GetIpPingKey(node.Id, ipIsp.Ip)
	//		if lossIpsMap[key] {
	// TODO
	return !f.rttIpsMap[ip.Ip]
}

func (f *RttFilter) Name() string {
	return "Rtt"
}

type IpFilterPrivate struct {
	Switch
}

func (f *IpFilterPrivate) Filter(ip *public.RtIpStatus) bool {
	return !publicUtil.IsPrivateIP(ip.Ip)
}

func (f *IpFilterPrivate) Name() string {
	return "PrivateIp"
}

type IpFilterForbidden struct {
	Switch
}

func (f *IpFilterForbidden) Filter(ip *public.RtIpStatus) bool {
	return !ip.Forbidden
}

func (f *IpFilterForbidden) Name() string {
	return "IpFrobidden"
}

type IpFilterProbeSpeed struct {
	Switch
}

func (f *IpFilterProbeSpeed) Filter(ip *public.RtIpStatus) bool {
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

func (f *IpFilterIpv6) Filter(ip *public.RtIpStatus) bool {
	return publicUtil.IsIPv6(ip.Ip)
}

func (f *IpFilterIpv6) Name() string {
	return "Ipv6"
}

type IpFilterTcpRetrans struct {
	Switch
}

func (f *IpFilterTcpRetrans) Filter(ip *public.RtIpStatus) bool {
	// TODO
	return true
}

func (f *IpFilterTcpRetrans) Name() string {
	return "TcpRetrans"
}

type IpFilterAvailableBw struct {
	Switch
}

func (f *IpFilterAvailableBw) Filter(ip *public.RtIpStatus) bool {
	// TODO
	return true
}

func (f *IpFilterAvailableBw) Name() string {
	return "AvailableBw"
}

type Switch struct {
	enable bool
}

func (s *Switch) Enable() bool {
	return s.enable
}

type SwitchConf struct {
	Dynamic        bool
	NotBanProv     bool
	Serving        bool
	NoStreamdPorts bool
	TimeLimit      bool
	Abilities      bool
	Services       bool
	PrivateIp      bool
	Forbidden      bool
	ProbeSpeed     bool
	Ipv6           bool
	Rtt            bool
	Loss           bool
	TcpRetrans     bool
	AvailableBw    bool
}

func NewFilterMgr(filterType string, conf SwitchConf) *FilterMgr {
	nodeFilters := []NodeFilter{
		&AbilitiesFilter{
			Switch: Switch{
				enable: conf.Abilities,
			},
		},
		&ServicesFilter{
			Switch: Switch{
				enable: conf.Services,
			},
		},
		&DynamicFilter{
			Switch: Switch{
				enable: conf.Dynamic,
			},
		},
		&NotBanProvFilter{
			Switch: Switch{
				enable: conf.NotBanProv,
			},
		},
		&ServingFilter{
			Switch: Switch{
				enable: conf.Serving,
			},
		},
		&NoStreamdPortsFilter{
			Switch: Switch{
				enable: conf.NoStreamdPorts,
			},
		},
		&TimeLimitFilter{
			Switch: Switch{
				enable: conf.TimeLimit,
			},
		},
	}
	ipFilters := []IpFilter{
		&IpFilterPrivate{
			Switch: Switch{
				enable: conf.PrivateIp,
			},
		},
		&IpFilterForbidden{
			Switch: Switch{
				enable: conf.Forbidden,
			},
		},
		&IpFilterProbeSpeed{
			Switch: Switch{
				enable: conf.ProbeSpeed,
			},
		},
		&IpFilterIpv6{
			Switch: Switch{
				enable: conf.Ipv6,
			},
		},
		&RttFilter{
			Switch: Switch{
				enable: conf.Ipv6,
			},
		},
		&LossFilter{
			Switch: Switch{
				enable: conf.Ipv6,
			},
		},
		&IpFilterTcpRetrans{
			Switch: Switch{
				enable: conf.TcpRetrans,
			},
		},
		&IpFilterAvailableBw{
			Switch: Switch{
				enable: conf.AvailableBw,
			},
		},
	}
	return &FilterMgr{
		nodeFilters:      nodeFilters,
		ipFilters:        ipFilters,
		statistics:       make(map[string]int),
		nodeAvailability: make(map[string]*commonUtil.NodeAvailabilityInfo),
		filterType:       filterType,
	}
}

func (m *FilterMgr) FilterNode(node *public.RtNode) bool {
	if m.filterType == FilterTypeMongo {
		return m.FilterNodeByAvailability(node)
	}
	for _, filter := range m.nodeFilters {
		if !filter.Enable() {
			continue
		}
		if !filter.Filter(node) {
			m.statistics[filter.Name()]++
			return false
		}
	}
	return true
}

func (m *FilterMgr) FilterIp(node *public.RtNode, ip *public.RtIpStatus) bool {
	if m.filterType == FilterTypeMongo {
		return m.FilterIpByAvailability(node, ip)
	}
	for _, filter := range m.ipFilters {
		if !filter.Enable() {
			continue
		}
		if !filter.Filter(ip) {
			m.statistics[filter.Name()]++
			return false
		}
	}
	return true
}

func (m *FilterMgr) FilterNodeByAvailability(node *public.RtNode) bool {
	if !node.IsDynamic {
		return false
	}
	if node.IsBanTransProv {
		return false
	}
	nodeInfo, ok := m.nodeAvailability[node.Id]
	if !ok {
		return true
	}
	if !nodeInfo.IsNodePass {
		m.statistics[nodeInfo.ExcludeReason]++
		return false
	}
	return true
}

func (m *FilterMgr) FilterIpByAvailability(node *public.RtNode, ip *public.RtIpStatus) bool {
	nodeInfo, ok := m.nodeAvailability[node.Id]
	if !ok {
		return true
	}
	ipInfo, ok := nodeInfo.IpAvailability[ip.Ip]
	if !ok {
		return true
	}
	if !ipInfo.IsIPPass {
		m.statistics[ipInfo.ExcludeReason]++
		return false
	}
	return true
}

func (m *FilterMgr) GetStatistics() map[string]int {
	return m.statistics
}

func (m *FilterMgr) SetResources(resources resources.Resources) {
	m.resources = resources
}

func (m *FilterMgr) LoadFilterData() {
	result, err := m.resources.NodeFilterCol.GetLatestAvailableResource(context.Background(), public.QualityLevelLow)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("LoadFilterData, result len:", len(result))
	for _, resource := range result {
		nodeInfo := &commonUtil.NodeAvailabilityInfo{
			NodeId:         resource.NodeId,
			IsNodePass:     resource.IsNodePass,
			ExcludeReason:  resource.ExcludeReason,
			IpAvailability: make(map[string]commonUtil.IpAvailabilityInfo),
		}

		for _, ip := range resource.Ips {
			nodeInfo.IpAvailability[ip.Ip] = commonUtil.IpAvailabilityInfo{
				Ip:            ip.Ip,
				IsIPPass:      ip.IsIPPass,
				ExcludeReason: ip.ExcludeReason,
			}
		}

		m.nodeAvailability[resource.NodeId] = nodeInfo
	}
}

func (m *FilterMgr) SetFilterType(filterType string) {
	m.filterType = filterType
}

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
}

type IpFilter interface {
	Filter(ip *public.RtIpStatus) bool
	Name() string
}

type FilterMgr struct {
	nodeFilters      []NodeFilter
	ipFilters        []IpFilter
	statistics       map[string]int
	resources        resources.Resources
	nodeAvailability map[string]*commonUtil.NodeAvailabilityInfo
}

type DynamicFilter struct {
}

func (f *DynamicFilter) Filter(node *public.RtNode) bool {
	return node.IsDynamic
}

func (f *DynamicFilter) Name() string {
	return "Dynamic"
}

type NotBanProvFilter struct {
}

func (f *NotBanProvFilter) Filter(node *public.RtNode) bool {
	return !node.IsBanTransProv
}

func (f *NotBanProvFilter) Name() string {
	return "NotBanProv"
}

type ServingFilter struct {
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

type TimeLimitFilter struct {
}

func (f *TimeLimitFilter) Filter(node *public.RtNode) bool {
	return len(node.Schedules) == 0
}

func (f *TimeLimitFilter) Name() string {
	return "TimeLimit"
}

type IpFilterPrivate struct {
}

func (f *IpFilterPrivate) Filter(ip *public.RtIpStatus) bool {
	return !publicUtil.IsPrivateIP(ip.Ip)
}

func (f *IpFilterPrivate) Name() string {
	return "PrivateIp"
}

type IpFilterForbidden struct {
}

func (f *IpFilterForbidden) Filter(ip *public.RtIpStatus) bool {
	return !ip.Forbidden
}

func (f *IpFilterForbidden) Name() string {
	return "IpFrobidden"
}

type IpFilterProbeSpeed struct {
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
}

func (f *IpFilterIpv6) Filter(ip *public.RtIpStatus) bool {
	return publicUtil.IsIPv6(ip.Ip)
}

func (f *IpFilterIpv6) Name() string {
	return "Ipv6"
}

func NewFilterMgr() *FilterMgr {
	nodeFilters := []NodeFilter{
		&DynamicFilter{},
		&NotBanProvFilter{},
		&ServingFilter{},
		&NoStreamdPortsFilter{},
		&TimeLimitFilter{},
	}
	ipFilters := []IpFilter{
		&IpFilterPrivate{},
		&IpFilterForbidden{},
		&IpFilterProbeSpeed{},
		&IpFilterIpv6{},
	}
	return &FilterMgr{
		nodeFilters:      nodeFilters,
		ipFilters:        ipFilters,
		statistics:       make(map[string]int),
		nodeAvailability: make(map[string]*commonUtil.NodeAvailabilityInfo),
	}
}

func (m *FilterMgr) FilterNode(node *public.RtNode) bool {
	for _, filter := range m.nodeFilters {
		if !filter.Filter(node) {
			m.statistics[filter.Name()]++
			return false
		}
	}
	return true
}

func (m *FilterMgr) FilterIp(ip *public.RtIpStatus) bool {
	for _, filter := range m.ipFilters {
		if !filter.Filter(ip) {
			m.statistics[filter.Name()]++
			return false
		}
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
	for _, resource := range result {
		/*
			nodeptr := s.GetNodePtr(resource.NodeId)
			machineid := resource.NodeId
			if nodeptr != nil && len(nodeptr.MachineId) > 0 {
				machineid = nodeptr.MachineId
			}
		*/

		nodeInfo := &commonUtil.NodeAvailabilityInfo{
			NodeId: resource.NodeId,
			//MachineId:      machineid,
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

package nodemgr

import (
	public "github.com/qbox/mikud-live/common/model"
)

type NodeFilter interface {
	Filter(node *public.RtNode) bool
	Name() string
	Enable() bool
	SetEnable(enable bool)
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

type NotFilterNat1 struct {
	Switch
}

func (f *NotFilterNat1) Filter(node *public.RtNode) bool {
	return !node.IsNat1()
}

func (f *NotFilterNat1) Name() string {
	return "NotNat1"
}

type NodeFilterStatic struct {
	Switch
}

func (f *NodeFilterStatic) Filter(node *public.RtNode) bool {
	return !node.IsDynamic
}

func (f *NodeFilterStatic) Name() string {
	return "Static"
}

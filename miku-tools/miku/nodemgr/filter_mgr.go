package nodemgr

import (
	"context"
	"errors"
	"log"
	"mikutool/config"
	"mikutool/resources"
	"sort"

	commonUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/cmd/sched/dal"
	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/rs/zerolog"
)

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
	eventHandlers    []EventHandler
	allNodes         []*public.RtNode
	conf             *config.Config
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

type EventHandler interface {
	EventHandler(any)
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
	eventHandlers := []EventHandler{}
	for _, filter := range ipFilters {
		if f, ok := filter.(EventHandler); ok {
			eventHandlers = append(eventHandlers, f)
		}
	}
	for _, filter := range nodeFilters {
		if f, ok := filter.(EventHandler); ok {
			eventHandlers = append(eventHandlers, f)
		}
	}
	return &FilterMgr{
		nodeFilters:      nodeFilters,
		ipFilters:        ipFilters,
		statistics:       make(map[string]int),
		nodeAvailability: make(map[string]*commonUtil.NodeAvailabilityInfo),
		filterType:       filterType,
		eventHandlers:    eventHandlers,
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
		if !filter.Filter(node, ip) {
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
	for _, handler := range m.eventHandlers {
		handler.EventHandler(resources)
	}
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

func (m *FilterMgr) setTcpRetransMap(tcpRetransMap map[string]float64) {
	for _, filter := range m.ipFilters {
		if f, ok := filter.(*IpFilterTcpRetrans); ok {
			f.SetData(tcpRetransMap)
			return
		}
	}
}

func (m *FilterMgr) SetAllNodes(allNodes []*public.RtNode) {
	m.allNodes = allNodes
	tcpRetranMap, err := dal.GetAllNodesTcpRetranInfo(zerolog.Logger{}, m.resources.Redis, allNodes)
	if err != nil {
		log.Println("SetAllNodes GetAllNodesTcpRetranInfo err", err)
		return
	}
	tcpRetranPreparedIPs, err := m.prepareTCPRetranFilter(allNodes, tcpRetranMap)
	if err != nil {
		log.Println("SetAllNodes prepareTCPRetranFilter err", err)
		return
	}
	m.setTcpRetransMap(tcpRetranPreparedIPs)
}

func (m *FilterMgr) prepareTCPRetranFilter(nodes []*public.RtNode, tcpRetranMap map[string]*public.TcpRetranInfo) (map[string]float64, error) {
	// 检查是否有节点重传率信息和节点信息
	if len(nodes) == 0 || len(tcpRetranMap) == 0 || tcpRetranMap == nil {
		return nil, errors.New("nodes or tcpRetranMap is nil")
	}

	type nodeRetran struct {
		node *public.RtNode
		rate float64
		ips  []string
	}

	// 收集所有需要过滤的节点信息
	nodeInfos := make([]nodeRetran, 0, len(nodes))
	for _, node := range nodes {
		if info, ok := tcpRetranMap[node.Id]; ok && info != nil {
			if info.AverageRate < m.conf.TcpRetranFilterConfig.TCPRetranRateThreshold {
				continue
			}
			validIPs := make([]string, 0, len(node.Ips))
			for _, ip := range node.Ips {
				if !ip.Forbidden && !publicUtil.IsPrivateIP(ip.Ip) {
					validIPs = append(validIPs, ip.Ip)
				}
			}
			if len(validIPs) > 0 {
				nodeInfos = append(nodeInfos, nodeRetran{
					node: node,
					rate: info.AverageRate,
					ips:  validIPs,
				})
			}
		}
	}

	// 计算总的不合格 IP 数量
	totalUnqualifiedIPs := 0
	for _, nodeInfo := range nodeInfos {
		for range nodeInfo.ips {
			totalUnqualifiedIPs += len(nodeInfo.ips)
		}
	}

	// 按照重传率排序
	sort.Slice(nodeInfos, func(i, j int) bool {
		return nodeInfos[i].rate > nodeInfos[j].rate
	})

	// 初始化待过滤 IP 列表
	tcpRetranPreparedIPs := make(map[string]float64, m.conf.TcpRetranFilterConfig.MaxFilterIPs+1)

	// 选择重传率最高的节点进行过滤
	totalPreparedIPs := 0
	for i := 0; i < len(nodeInfos) && i < m.conf.TcpRetranFilterConfig.MaxFilterNodes; i++ {
		for _, ip := range nodeInfos[i].ips {
			tcpRetranPreparedIPs[ip] = nodeInfos[i].rate
			totalPreparedIPs++
			// 如果达到最大过滤 IP 数量，退出
			if totalPreparedIPs >= m.conf.TcpRetranFilterConfig.MaxFilterIPs {
				break
			}
		}
		// 如果达到最大过滤 IP 数量，退出
		if totalPreparedIPs >= m.conf.TcpRetranFilterConfig.MaxFilterIPs {
			break
		}
	}

	log.Println("prepareTCPRetranFilter, tcpRetranPreparedIPs len:",
		len(tcpRetranPreparedIPs), "totalPreparedIPs:", totalPreparedIPs,
		"totalUnqualifiedIPs:", totalUnqualifiedIPs,
		"totalUnqualifiedNodes:", len(nodeInfos))

	return tcpRetranPreparedIPs, nil
}

func (m *FilterMgr) SetConf(conf *config.Config) {
	m.conf = conf
	for _, filter := range m.ipFilters {
		if f, ok := filter.(*IpFilterTcpRetrans); ok {
			f.SetConf(conf)
		}
	}
}

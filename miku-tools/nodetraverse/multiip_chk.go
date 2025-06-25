package nodetraverse

import (
	"middle-source-analysis/callback"

	public "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/rs/zerolog/log"
)

func RegisterMultiIPChk() {
	Register(&MultiIPChk{})
}

type MultiIPChk struct {
	cnt int
}

func (m *MultiIPChk) OnIp(node *public.RtNode, ip *public.RtIpStatus) {
}

func (m *MultiIPChk) OnNode(node *public.RtNode, ipParser *ipdb.City, callback callback.Callback) {
	if !node.IsDynamic {
		return
	}
	if node.IsNat1() {
		return
	}
	lastIsp := ""
	if len(node.Ips) > 3 {
		//log.Info().Msgf("node %s has multi ip, nodetype: %s, ipCnt: %d, cnt : %d", node.Id, node.ResourceType, len(node.Ips), m.cnt)
		m.cnt++
		for _, ip := range node.Ips {
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			if ip.IsIPv6 {
				continue
			}

			locate, err := ipParser.Find(ip.Ip)
			if err != nil {
				log.Error().Msgf("node %s machineId : %s, ip: %s, err: %+v", node.Id, node.MachineId, ip.Ip, err)
				return
			}

			if lastIsp != "" && lastIsp != locate.Isp {
				log.Info().Msgf("node %s machineId : %s, has multi ip, nodetype: %s, ipCnt: %d, cnt : %d, ip: %s, isp: %s, lastIsp: %s",
					node.Id, node.MachineId, node.ResourceType, len(node.Ips), m.cnt, ip.Ip, locate.Isp, lastIsp)
			}
			lastIsp = locate.Isp
		}
	}
}

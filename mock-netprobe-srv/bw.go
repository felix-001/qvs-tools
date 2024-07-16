package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	publicUtil "github.com/qbox/mikud-live/common/util"
)

func (s *NetprobeSrv) FillAreaBw(areaIsp string) string {
	for i, node := range s.nodes {
		for j, ip := range node.Ips {
			areaIsp_ := s.getIpAreaIsp(ip.Ip)
			if areaIsp_ == areaIsp {
				s.nodes[i].Ips[j].OutMBps = ip.MaxOutMBps
			}
		}
	}
	return "success"
}

func (s *NetprobeSrv) FillBw(paramMap map[string]string) string {
	nodeId := paramMap["node"]
	fillType := paramMap["type"]
	for i, node := range s.nodes {
		if node.Id == nodeId {
			for j, ip := range node.Ips {
				if fillType == "in" {
					if ip.MaxInMBps == 0 {
						continue
					}
					s.nodes[i].Ips[j].InMBps = ip.MaxInMBps * 0.9
					log.Printf("ip: %s InMBps: %.1f\n", ip.Ip, s.nodes[i].Ips[j].InMBps)
				} else {
					if ip.MaxOutMBps == 0 {
						continue
					}
					s.nodes[i].Ips[j].OutMBps = ip.MaxOutMBps * 0.9
					log.Printf("ip: %s OutMBps: %.1f\n", ip.Ip, s.nodes[i].Ips[j].OutMBps)
				}
			}
			break
		}
	}
	return "success"
}

func (s *NetprobeSrv) FillIspBw(isp string) string {
	for i, node := range s.nodes {
		for j, ip := range node.Ips {
			if ip.IpIsp.Isp == isp {
				log.Println("clear bw", node.Id, isp)
				if ip.MaxInMBps == 0 {
					s.nodes[i].Ips[j].MaxOutMBps = 10
				}
				s.nodes[i].Ips[j].OutMBps = ip.MaxOutMBps
			}
		}
	}
	return "success"
}

func (s *NetprobeSrv) ClearBw(nodeId string) string {
	for i, node := range s.nodes {
		if node.Id == nodeId {
			for j, _ := range node.Ips {
				s.nodes[i].Ips[j].OutMBps = 0
				s.nodes[i].Ips[j].InMBps = 0
			}
			break
		}
	}
	return "success"
}

func (s *NetprobeSrv) FillOutBw(nodeId string) string {
	for i, node := range s.nodes {
		if node.Id == nodeId {
			for j, ip := range node.Ips {
				if ip.MaxOutMBps == 0 {
					continue
				}
				s.nodes[i].Ips[j].OutMBps = ip.MaxOutMBps - 10
			}
			break
		}
	}
	return "success"
}

func (s *NetprobeSrv) FillInBw(nodeId string) string {
	for i, node := range s.nodes {
		if node.Id == nodeId {
			for j, ip := range node.Ips {
				if ip.MaxInMBps == 0 {
					continue
				}
				s.nodes[i].Ips[j].InMBps = ip.MaxInMBps * 0.8
			}
			break
		}
	}
	return "success"
}

func (s *NetprobeSrv) CostBw(nodeId string) {
	log.Println(len(s.nodes))
	for i, node := range s.nodes {
		if node.Id == nodeId {
			log.Println("found the node")
			for j, ip := range node.Ips {
				s.nodes[i].Ips[j].InMBps = ip.MaxInMBps * 0.7
			}
			return
		}
	}
	log.Println("node not found:", nodeId)
}

func (s *NetprobeSrv) GetAreaIspRootBwInfo(areaIsp string) string {
	res, err := s.redisCli.HGet(context.Background(), "miku_dynamic_root_nodes_map_douyu", areaIsp).Result()
	if err != nil {
		return fmt.Sprintf("err: %+v", err)
	}
	var rootNodeIds []string
	if err := json.Unmarshal([]byte(res), &rootNodeIds); err != nil {
		return fmt.Sprintf("err: %+v", err)
	}
	nodeCnt := 0
	out := ""
	for _, nodeId := range rootNodeIds {
		for _, node := range s.nodes {
			if node.Id != nodeId {
				continue
			}
			var inBw, outBw, maxInBw, maxOutBw float64
			for _, ip := range node.Ips {
				if ip.IsIPv6 {
					continue
				}
				if publicUtil.IsPrivateIP(ip.Ip) {
					continue
				}
				inBw += ip.InMBps * 8
				outBw += ip.OutMBps * 8
				maxInBw += ip.MaxInMBps * 8
				maxOutBw += ip.MaxOutMBps * 8
			}
			out += fmt.Sprintf("node: %s inMpbs: %.0f maxInMbps: %.0f inRatio: %.3f outMbps: %.0f maxOutMbps: %.0f outRatio: %.3f\n",
				node.Id, inBw, maxInBw, inBw/maxInBw, outBw, maxOutBw, outBw/maxOutBw)
			nodeCnt++
		}
	}
	out += fmt.Sprintf("node count: %d\n", nodeCnt)

	return out
}

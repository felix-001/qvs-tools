package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	public "github.com/qbox/mikud-live/common/model"
	schedutil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func (s *Parser) GetNodeByIp() {
	for _, node := range s.allNodesMap {
		for _, ipInfo := range node.Ips {
			if ipInfo.Ip == s.conf.Ip {
				_, ok := s.allRootNodesMapByNodeId[node.Id]
				fmt.Println("nodeId:", node.Id, "machineId:", node.MachineId, "isRoot:", ok)
				break
			}
		}
	}
}

func (s *Parser) NodeDis() {
	areaMap := make(map[string]bool)
	for _, node := range s.allNodesMap {
		if node.RuntimeStatus != "Serving" {
			continue
		}
		if node.StreamdPorts.Http == 0 {
			continue
		}
		if node.StreamdPorts.Rtmp == 0 {
			continue
		}
		if !util.CheckNodeUsable(zlog.Logger, node, consts.TypeLive) {
			continue
		}
		isp, area, _ := getNodeLocate(node, s.IpParser)
		areaMap[area+isp] = true

	}

	needAreas := make([]string, 0)
	for _, area := range Areas {
		for _, isp := range Isps {
			areaIsp := area + isp
			if _, ok := areaMap[areaIsp]; !ok {
				needAreas = append(needAreas, areaIsp)
			}
		}
	}
	bytes, err := json.MarshalIndent(areaMap, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
	fmt.Println("needAreas:", needAreas)
}

type NodeCallback interface {
	OnNode(node *public.RtNode)
	OnIp(node *public.RtNode, ip *public.RtIpStatus)
}

type NodeTraverse struct {
	RedisAddrs []string
	logger     zerolog.Logger
	cb         NodeCallback
}

func NewNodeTraverse(logger zerolog.Logger, cb NodeCallback, RedisAddrs []string) *NodeTraverse {
	return &NodeTraverse{
		logger:     logger,
		cb:         cb,
		RedisAddrs: RedisAddrs,
	}
}

func (n *NodeTraverse) Traverse() {
	fmt.Println("NodeTraverse")
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      n.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		zlog.Fatal().Err(err).Msg("")
	}

	allNodes, err := public.GetAllRTNodes(n.logger, redisCli)
	if err != nil {
		n.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}
	for _, node := range allNodes {
		n.cb.OnNode(node)
		for _, ip := range node.Ips {
			n.cb.OnIp(node, &ip)
		}
	}
}

// Item 结构体对应 JSON 中的顶级对象
type StaticNode struct {
	ID             string          `json:"id"`
	SreID          string          `json:"sreId"`
	NodeID32       int             `json:"nodeId32"`
	IDC            string          `json:"idc"`
	IDCTitle       string          `json:"idcTitle"`
	Host           string          `json:"host"`
	Status         string          `json:"status"`
	BWMbps         int             `json:"bwMbps"`
	IPs            []string        `json:"ips"`
	LanIP          string          `json:"lanIP"`
	RtIPs          []RtIP          `json:"rtIps"`
	OutwardIP      string          `json:"outwardIP"`
	CPU            CPUInfo         `json:"cpu"`
	Services       Services        `json:"services"`
	MemInfo        MemInfo         `json:"mem_info"`
	Disks          map[string]Disk `json:"disks"`
	DisabledTill   string          `json:"disabledTill"`
	LastReport     string          `json:"lastReport"`
	LastCodec      string          `json:"lastCodec"`
	Provider       string          `json:"provider"`
	Comment        string          `json:"comment"`
	ChargeZone     string          `json:"chargeZone"`
	Date           int             `json:"date"`
	Groups         []string        `json:"groups"`
	GroupWithouts  []string        `json:"groupWithouts"`
	Abilities      Abilities       `json:"abilities"`
	OnlineTime     int             `json:"onlineTime"`
	Runtime        string          `json:"runtime"`
	AbnormalFields []interface{}   `json:"abnormalFields"`
}

// RtIP 结构体对应 rtIps 数组中的元素
type RtIP struct {
	IP           string        `json:"ip"`
	ISP          string        `json:"isp"`
	ResolveIsps  []string      `json:"resolveIsps,omitempty"`
	InMbps       float64       `json:"inMbps"`
	OutMbps      float64       `json:"outMbps"`
	InMBpsLimit  float64       `json:"inMBpsLimit"`
	OutMBpsLimit float64       `json:"outMBpsLimit"`
	AnalysisIPs  []interface{} `json:"analysisIPs,omitempty"`
	ResolveAll   bool          `json:"resolveAll,omitempty"`
}

// CPUInfo 结构体对应 cpu 字段
type CPUInfo struct {
	CoreNumber int     `json:"coreNumber"`
	Load       CPULoad `json:"load"`
	Idle       float64 `json:"idle"`
}

// CPULoad 结构体对应 cpu.load 字段
type CPULoad struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// ServiceInfo 结构体对应 services 中的每个服务
type ServiceInfo struct {
	CPU             int    `json:"cpu"`
	RSS             int    `json:"rss"`
	FD              int    `json:"fd"`
	Uptime          int    `json:"uptime"`
	Version         string `json:"version"`
	FVersion        string `json:"fversion"`
	PandoraFilesCnt int    `json:"pandora_files_cnt"`
}

// Services 结构体对应 services 字段，是一个 map
type Services map[string]ServiceInfo

// MemInfo 结构体对应 mem_info 字段
type MemInfo struct {
	MemTotal int `json:"mem_total"`
	MemFree  int `json:"mem_free"`
	Buffers  int `json:"buffers"`
	Cached   int `json:"cached"`
	Used     int `json:"used"`
}

// Disk 结构体对应 disks 中的每个磁盘信息
type Disk struct {
	Device string `json:"device"`
	Size   int    `json:"size"`
	Used   int    `json:"used"`
}

// Ability 结构体对应 abilities 中的每个能力
type Ability struct {
	Can    bool `json:"can"`
	Frozen bool `json:"frozen"`
}

// Abilities 结构体对应 abilities 字段，是一个 map
type Abilities struct {
	Acc              Ability `json:"acc"`
	Mid              Ability `json:"mid"`
	Ping             Ability `json:"ping"`
	Play             Ability `json:"play"`
	PubSimplifyMulti Ability `json:"pub_simplify_multi"`
	Publish          Ability `json:"publish"`
	Rtmpgate         Ability `json:"rtmpgate"`
}

func (s *Parser) loadStaticNodes() []StaticNode {
	var nodes struct {
		StaticNodes []StaticNode `json:"items"`
	}
	if s.conf.F == "" {
		log.Fatalf("load config failed, err: %v", "no config file")
	}
	if err := qconfig.LoadFile(&nodes, s.conf.F); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	log.Println("load config success, len:", len(nodes.StaticNodes))
	return nodes.StaticNodes
}

func (s *Parser) getQvsStaticNodes() {
	rawNodes := s.loadStaticNodes()
	result := make([]*StaticNode, 0)
	notCan := 0
	frozen := 0
	abnorml := 0
	for _, node := range rawNodes {
		if node.Runtime == "abnormal" {
			abnorml++
		}
		if !node.Abilities.PubSimplifyMulti.Can {
			notCan++
			//fmt.Println(node.ID)
			continue
		}
		if node.Abilities.PubSimplifyMulti.Frozen {
			frozen++
			continue
		}
		result = append(result, &node)
	}
	log.Println("qvs static nodes:", len(result), "notCan:", notCan, "frozen:", frozen, "abnorml:", abnorml)
	s.probeStaticNodeTalk(result)
}

func (s *Parser) probeStaticNodeTalk(nodes []*StaticNode) {
	ipParser, err := ipdb.NewCity(s.conf.IPDB)
	if err != nil {
		log.Fatalf("[IPDB NewCity] err: %+v\n", err)
	}
	for _, node := range nodes {
		for _, ip := range node.RtIPs {
			if schedutil.IsPrivateIP(ip.IP) {
				continue
			}
			loc, err := ipParser.Find(ip.IP)
			if err != nil {
				log.Printf("查找IP %s 的位置信息失败: %v", ip.IP, err)
				continue
			}
			if loc.Region == "香港" {
				continue
			}
			if loc.Country != "中国" {
				log.Println(loc.Country, loc.Region, loc.City, loc.Isp, ip.IP)
				continue
			}
			if IsIpv6(ip.IP) {
				continue
			}
			log.Println(loc.Country, loc.Region, loc.City, loc.Isp, ip.IP)
			httpUrl, httpsUrl := FormatTalkAddr("1000000000000000", ip.IP, 10000)

			// 创建一个带有10秒超时的上下文
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// 创建一个新的http客户端
			client := &http.Client{}

			// 发送HTTP请求
			reqHTTP, err := http.NewRequestWithContext(ctx, "GET", httpUrl, nil)
			if err != nil {
				log.Printf("创建HTTP请求失败: %v", err)
			} else {
				respHTTP, err := client.Do(reqHTTP)
				if err != nil {
					log.Printf("发送HTTP请求失败: %v", err)
				} else {
					defer respHTTP.Body.Close()
					// 读取HTTP响应的body
					bodyHTTP, err := io.ReadAll(respHTTP.Body)
					if err != nil {
						log.Printf("读取HTTP响应body失败: %v", err)
					} else {
						log.Printf("HTTP响应body: %s", string(bodyHTTP))
					}
				}
			}

			// 发送HTTPS请求
			reqHTTPS, err := http.NewRequestWithContext(ctx, "GET", httpsUrl, nil)
			if err != nil {
				log.Printf("创建HTTPS请求失败: %v", err)
			} else {
				respHTTPS, err := client.Do(reqHTTPS)
				if err != nil {
					log.Printf("发送HTTPS请求失败: %v", err)
				} else {
					defer respHTTPS.Body.Close()
					// 读取HTTPS响应的body
					bodyHTTPS, err := io.ReadAll(respHTTPS.Body)
					if err != nil {
						log.Printf("读取HTTPS响应body失败: %v", err)
					} else {
						log.Printf("HTTPS响应body: %s", string(bodyHTTPS))
					}
				}
			}
		}
	}
}

func IP4ToUint(ip string) uint32 {

	p := net.ParseIP(ip)
	return binary.BigEndian.Uint32(p.To4())
}
func IP4ToUintStr(ip string, base int) string {
	u := IP4ToUint(ip)
	return strconv.FormatUint(uint64(u), base)
}

func FormatTalkAddr(gbId, rtpIp string, ssrc uint32) (string, string) {

	host := IP4ToUintStr(rtpIp, 10) + ".cloudvdn.com"
	httpUrl := fmt.Sprintf("http://%s/api/v1/gb28181?action=append_audio_pcm&id=%s&ssrc=%d",
		host, gbId, ssrc)
	httpsUrl := fmt.Sprintf("https://%s/api/v1/gb28181?action=append_audio_pcm&id=%s&ssrc=%d",
		host, gbId, ssrc)

	return httpUrl, httpsUrl
}

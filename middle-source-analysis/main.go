package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/cmd/sched/dal"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
	"github.com/redis/go-redis/v9"
)

const (
	confFile = "./middle-source.json"
)

type Config struct {
	RedisAddrs []string    `json:"redis_addrs"`
	IPDB       ipdb.Config `json:"ipdb"`
}

type Parser struct {
	redisCli                 *redis.ClusterClient
	ipParser                 *ipdb.City
	nodeStremasMap           map[string]*model.NodeStreamInfo
	allNodesMap              map[string]*model.RtNode
	allRootNodesMapByAreaIsp map[string][]*DynamicRootNode
	allRootNodesMapByNodeId  map[string]*model.RtNode
	streamDetailMap          map[string]map[string]map[string]*StreamInfo
}

func newParser(conf *Config) *Parser {
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})
	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	ipParser, err := ipdb.NewCity(conf.IPDB)
	if err != nil {
		log.Fatalf("[IPDB NewCity] err: %+v\n", err)
	}
	return &Parser{redisCli: redisCli, ipParser: ipParser}
}

func (s *Parser) getNodeAllStreams(nodeId string) (*model.NodeStreamInfo, error) {
	ctx := context.Background()
	val, err := s.redisCli.Get(ctx, util.GetStreamReportRedisKey(nodeId)).Result()
	if err != nil {
		return nil, err
	}
	var nodeStreamInfo model.NodeStreamInfo
	if err = json.Unmarshal([]byte(val), &nodeStreamInfo); err != nil {
		log.Printf("[GetNodeStreams][Unmarshal], nodeId:%s, value:%s\n", nodeId, val)
		return nil, err
	}
	return &nodeStreamInfo, nil
}

type DynamicRootNode struct {
	NodeId        string
	Forbidden     bool
	Err           string
	ForbiddenTime int64
}

func GetDynamicRootNodes(redisCli *redis.ClusterClient) (
	map[string][]*DynamicRootNode, error) {

	dynamicRootNodesMap := make(map[string][]*DynamicRootNode)
	ctx := context.Background()
	res, err := redisCli.HGetAll(ctx, "miku_dynamic_root_nodes_map").Result()
	if err != nil {
		log.Println(err)
		return dynamicRootNodesMap, consts.ErrRedisHGetAll
	}

	for areaIsp, value := range res {
		var nodes []*DynamicRootNode
		if err = json.Unmarshal([]byte(value), &nodes); err != nil {
			log.Println(err)
			continue
		}

		dynamicRootNodesMap[areaIsp] = nodes
	}

	return dynamicRootNodesMap, nil
}

func (s *Parser) buildAllNodesMap() {
	allNodes, err := dal.GetAllNode(s.redisCli)
	if err != nil {
		log.Fatalln(err)
	}
	allNodesMap := make(map[string]*model.RtNode)
	for _, node := range allNodes {
		allNodesMap[node.Id] = node
	}
	s.allNodesMap = allNodesMap
}

func (s *Parser) buildNodeStreamsMap() {
	nodeStreamsMap := make(map[string]*model.NodeStreamInfo)
	for nodeId := range s.allNodesMap {
		report, err := s.getNodeAllStreams(nodeId)
		if err != nil || report == nil {
			continue
		}
		nodeStreamsMap[nodeId] = report
	}
	s.nodeStremasMap = nodeStreamsMap
	log.Println("nodeStremasMap len", len(s.nodeStremasMap))
}

func (s *Parser) getNodeOnlineNum(streamInfo *model.StreamInfoRT) int {
	totalOnlineNum := 0
	for _, player := range streamInfo.Players {
		for _, ipInfo := range player.Ips {
			totalOnlineNum += int(ipInfo.OnlineNum)
			log.Println("protocol:", player.Protocol)
		}
	}
	return totalOnlineNum
}

func getIpAreaIsp(ipParser *ipdb.City, ip string) (string, string, error) {
	locate, err := ipParser.Find(ip)
	if err != nil {
		return "", "", err
	}
	areaIspKey, _ := util.GetAreaIspKey(locate)
	parts := strings.Split(areaIspKey, "_")
	if len(parts) != 5 {
		return "", "", fmt.Errorf("parse areaIspKey err, %s", areaIspKey)
	}
	area := parts[3]
	isp := parts[4]
	if area == "" {
		return "", "", fmt.Errorf("area empty")
	}
	if isp == "" {
		return "", "", fmt.Errorf("isp empty")
	}
	return area, isp, nil
}

func (s *Parser) getStreamNodes(sid, bkt string) map[string][]*model.RtNode {
	nodeMap := make(map[string][]*model.RtNode)
	for areaIsp, nodes := range s.allRootNodesMapByAreaIsp {
		for _, rootNode := range nodes {
			report := s.nodeStremasMap[rootNode.NodeId]
			if time.Now().Unix()-report.LastUpdateTime > 300 {
				log.Println(rootNode.NodeId, "stream offline", time.Unix(report.LastUpdateTime, 0).Format("2006-01-02 15:04:05 -0700 MST"))
				continue
			}
			for _, stream := range report.Streams {
				if stream.Bucket == bkt && stream.Key == sid {
					onlineNum := s.getNodeOnlineNum(stream)
					log.Println("node:", rootNode.NodeId, "onlineNum:", onlineNum,
						"bandwidth:", stream.Bandwidth, "relayBandWidth", stream.RelayBandwidth,
						"relayType:", stream.RelayType)
					node, ok := s.allNodesMap[rootNode.NodeId]
					if !ok {
						log.Println("node", rootNode.NodeId, "not found in all nodes buf")
						continue
					}
					nodeMap[areaIsp] = append(nodeMap[areaIsp], node)
					break
				}
			}
		}
	}
	return nodeMap
}

func (s *Parser) dump() {
	for streamId, streamDetail := range s.streamDetailMap {
		for isp, detail := range streamDetail {
			for area, streamInfo := range detail {
				log.Println(streamId, isp, area, streamInfo.OnlineNum, streamInfo.Ratio, streamInfo.EdgeNodeNum, streamInfo.RootNodeNum)
			}
		}
	}
	/*
		log.Println("nodesMap len:", len(nodeMap))
		total := 0
		for areaIsp, nodes := range nodeMap {
			log.Println(areaIsp, ":", len(nodes))
			fmt.Printf("\t")
			for _, node := range nodes {
				fmt.Printf("%s ", node.Id)
			}
			total += len(nodes)
			fmt.Println("")
		}
		log.Println("total:", total)
	*/
}

func (s *Parser) buildRootNodesMap() {
	dynamicRootNodesMap, err := GetDynamicRootNodes(s.redisCli)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("map len", len(dynamicRootNodesMap))
	s.allRootNodesMapByNodeId = make(map[string]*model.RtNode)
	s.allRootNodesMapByAreaIsp = dynamicRootNodesMap
	for _, rootNodes := range dynamicRootNodesMap {
		for _, rootNode := range rootNodes {
			node, ok := s.allNodesMap[rootNode.NodeId]
			if !ok {
				log.Println("not found root node in all nodes buf", rootNode.NodeId)
				continue
			}
			s.allRootNodesMapByNodeId[rootNode.NodeId] = node
		}
	}
}

func (s *Parser) Run(sid, bkt string) {
	//nodeMap := s.getStreamNodes(sid, bkt)
	//s.dump(nodeMap)
	//s.buildBucketStreamsInfo()
}

type NodeDetail struct {
	OnlineNum uint32
	RelayBw   float64
	Bw        float64
	MaxBw     float64
	Ratio     float64
	RelayType uint32
	Protocol  string
	NodeId    string
}

type StreamInfo struct {
	EdgeNodeNum int
	EdgeNodes   []NodeDetail
	RootNodeNum int
	RootNodes   []NodeDetail
	RelayBw     float64
	Bw          float64
	OnlineNum   uint32
	Ratio       float64
}

func convertMbps(bw uint64) float64 {
	return float64(bw) * 8 / 1e6
}

// TODO: 节点的每个网卡的带宽利用率, 这里的出口带宽不能使用streamreport上报的，其他业务线，
// 或者别的bucket下的流也会使用这个出口带宽
func (s *Parser) getNodeDetailMap(streamDetail map[string]map[string]*StreamInfo,
	stream *model.StreamInfoRT, node *model.RtNode) {

	for _, player := range stream.Players {
		/*
			if len(player.Ips) > 1 {
				log.Println("player ips > 1", node.Id)
			}
		*/
		var lastIsp, lastArea string
		for _, ipInfo := range player.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				//log.Println("private ip", ipInfo.Ip)
				continue
			}
			if ipInfo.Ip == "" {
				//log.Println("ip empty")
				continue
			}
			area, isp, err := getIpAreaIsp(s.ipParser, ipInfo.Ip)
			if err != nil {
				log.Println("getIpAreaIsp err", ipInfo.Ip, err)
				continue
			}
			if lastIsp != "" && lastIsp != isp {
				log.Println("isp not equal", node.Id, stream.Key, "last:", lastIsp, "cur:", isp)
			}
			if lastArea != "" && lastArea != area {
				log.Println("area not equal", node.Id, stream.Key, "last:", lastArea, "cur:", area)
			}
			if lastIsp == "" {
				lastIsp = isp
			}
			if lastArea == "" {
				lastArea = area
			}
			if _, ok := streamDetail[isp]; !ok {
				streamDetail[isp] = make(map[string]*StreamInfo)
			}
			if _, ok := streamDetail[isp][area]; !ok {
				streamDetail[isp][area] = &StreamInfo{
					//RelayType: stream.RelayType,
					//Protocol:  player.Protocol,
					//RelayBw:   convertMbps(stream.RelayBandwidth),
					OnlineNum: ipInfo.OnlineNum,
					Bw:        convertMbps(ipInfo.Bandwidth),
				}
			}
			streamDetail[isp][area].OnlineNum += ipInfo.OnlineNum
			streamDetail[isp][area].Bw += float64(ipInfo.Bandwidth)
		}
	}
}

func (s *Parser) checkNodeStreamIpLocate(stream *model.StreamInfoRT, node *model.RtNode) {
	for _, player := range stream.Players {
		var lastIsp, lastArea string
		for _, ipInfo := range player.Ips {
			if publicUtil.IsPrivateIP(ipInfo.Ip) {
				//log.Println("private ip", ipInfo.Ip)
				continue
			}
			if ipInfo.Ip == "" {
				//log.Println("ip empty")
				continue
			}
			area, isp, err := getIpAreaIsp(s.ipParser, ipInfo.Ip)
			if err != nil {
				log.Println("getIpAreaIsp err", ipInfo.Ip, err)
				continue
			}
			if lastIsp != "" && lastIsp != isp {
				log.Println("isp not equal", node.Id, stream.Key, "last:", lastIsp, "cur:", isp)
			}
			if lastArea != "" && lastArea != area {
				log.Println("area not equal", node.Id, stream.Key, "last:", lastArea, "cur:", area)
			}
			if lastIsp == "" {
				lastIsp = isp
			}
			if lastArea == "" {
				lastArea = area
			}
		}
	}
}

func (s *Parser) getNodeStreaemDetail(stream *model.StreamInfoRT) (int, float64) {
	totalOnlineNum := 0
	var totalBw float64
	for _, player := range stream.Players {
		for _, ipInfo := range player.Ips {
			totalOnlineNum += int(ipInfo.OnlineNum)
			totalBw += convertMbps(ipInfo.Bandwidth)
		}
	}
	return totalOnlineNum, totalBw
}

func (s *Parser) check(node *model.RtNode, streamInfo *model.NodeStreamInfo) bool {
	if !node.IsDynamic {
		return false
	}
	if streamInfo == nil {
		//log.Println(node.Id, "not found stream info")
		return false
	}
	if time.Now().Unix()-streamInfo.LastUpdateTime > 300 {
		/*
			log.Println(node.Id, "stream offline",
				time.Unix(streamInfo.LastUpdateTime, 0).
					Format("2006-01-02 15:04:05 -0700 MST"))
		*/
		return false
	}
	if streamInfo.NodeId != node.Id {
		log.Println("check stream info node id err")
	}
	return true
}

func (s *Parser) calcRelayBw(streamDetail map[string]map[string]*StreamInfo, stream *model.StreamInfoRT, node *model.RtNode) {
	for _, detail := range streamDetail {
		for _, streamInfo := range detail {
			streamInfo.RelayBw += convertMbps(stream.RelayBandwidth)
		}
	}
}

func ContainInStringSlice(target string, slice []string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}

	return false
}

func (s *Parser) isRoot(node *model.RtNode) bool {
	_, ok := s.allRootNodesMapByNodeId[node.Id]
	return ok
}

func (s *Parser) dumpStreams(streamInfo *model.NodeStreamInfo) {
	for _, stream := range streamInfo.Streams {
		log.Printf("%+v", *stream)
	}
}

func getLocate(ip string, ipParser *ipdb.City) (string, string, string) {
	locate, err := ipParser.Find(ip)
	if err != nil {
		log.Println(err)
		return "", "", ""
	}
	if locate.Isp == "" {
		//log.Println("country", locate.Country, "isp", locate.Isp, "city", locate.City, "region", locate.Region, "ip", ip)
	}
	area := monitorUtil.ProvinceAreaRelation(locate.Region)
	return locate.Isp, area, locate.Region
}

func getNodeLocate(node *model.RtNode, ipParser *ipdb.City) (string, string) {
	for _, ip := range node.Ips {
		if ip.IsIPv6 {
			continue
		}
		if publicUtil.IsPrivateIP(ip.Ip) {
			continue
		}
		isp, area, _ := getLocate(ip.Ip, ipParser)
		if area != "" {
			return isp, area
		}
	}
	return "", ""
}

/*
流id - ISP - 大区
map[string]map[string]map[string]StreamInfo
*/
func (s *Parser) buildBucketStreamsInfo(bkt string) {
	s.streamDetailMap = make(map[string]map[string]map[string]*StreamInfo)
	for _, node := range s.allNodesMap {
		streamInfo := s.nodeStremasMap[node.Id]
		if !s.check(node, streamInfo) {
			continue
		}
		lastStream := ""
		isp, area := getNodeLocate(node, s.ipParser)
		if isp == "" || area == "" {
			//log.Println("node", node.Id, "get ip locate err")
			continue
		}
		for _, stream := range streamInfo.Streams {
			if stream.Bucket != bkt {
				continue
			}
			if lastStream != "" && stream.Key == lastStream {
				log.Println("two samle stream in one node", "nodeid:", node.Id, "streamid", stream.Key,
					"relayBandwidth:", stream.RelayBandwidth)
				s.dumpStreams(streamInfo)
			}
			if lastStream == "" {
				lastStream = stream.Key
			}
			if _, ok := s.streamDetailMap[stream.Key]; !ok {
				s.streamDetailMap[stream.Key] = make(map[string]map[string]*StreamInfo)
			}
			if _, ok := s.streamDetailMap[stream.Key][isp]; !ok {
				s.streamDetailMap[stream.Key][isp] = make(map[string]*StreamInfo)
			}
			onlineNum, bw := s.getNodeStreaemDetail(stream)
			isRoot := s.isRoot(node)
			if streamInfo, ok := s.streamDetailMap[stream.Key][isp][area]; !ok {
				s.streamDetailMap[stream.Key][isp][area] = &StreamInfo{
					OnlineNum: uint32(onlineNum),
					Bw:        bw,
					RelayBw:   convertMbps(stream.RelayBandwidth),
				}
				streamInfo := s.streamDetailMap[stream.Key][isp][area]
				if !isRoot {
					streamInfo.EdgeNodes = append(streamInfo.EdgeNodes, NodeDetail{
						NodeId: node.Id,
					})
				} else {
					streamInfo.RootNodes = append(streamInfo.RootNodes, NodeDetail{
						NodeId: node.Id,
					})
				}
			} else {
				streamInfo.OnlineNum += uint32(onlineNum)
				streamInfo.Bw += bw
				streamInfo.RelayBw += convertMbps(stream.RelayBandwidth)
				if !isRoot {
					streamInfo.EdgeNodes = append(streamInfo.EdgeNodes, NodeDetail{
						NodeId: node.Id,
					})
				} else {
					streamInfo.RootNodes = append(streamInfo.RootNodes, NodeDetail{
						NodeId: node.Id,
					})
				}
			}
		}
	}
	log.Println("total:", len(s.streamDetailMap))
}

func (s *Parser) calcRatioByStream() {
	for _, streamDetail := range s.streamDetailMap {
		for _, detail := range streamDetail {
			for _, streamInfo := range detail {
				streamInfo.Ratio = streamInfo.Bw / streamInfo.RelayBw
				streamInfo.EdgeNodeNum = len(streamInfo.EdgeNodes)
				streamInfo.RootNodeNum = len(streamInfo.RootNodes)
			}
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//sid := flag.String("sid", "", "流ID")
	bkt := flag.String("bkt", "douyu", "bucket ID")
	flag.Parse()
	if /**sid == "" || */ *bkt == "" {
		flag.PrintDefaults()
		return
	}
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	parser := newParser(&conf)
	parser.buildAllNodesMap()
	parser.buildNodeStreamsMap()
	parser.buildRootNodesMap()
	parser.buildBucketStreamsInfo(*bkt)
	parser.calcRatioByStream()
	parser.dump()
	//parser.Run(*sid, *bkt)

}

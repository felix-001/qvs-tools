package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/cmd/sched/dal"
	"github.com/qbox/mikud-live/common/model"
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
	redisCli       *redis.ClusterClient
	ipParser       *ipdb.City
	nodeStremasMap map[string]*model.NodeStreamInfo
	allNodesMap    map[string]*model.RtNode
	rootNodesMap   map[string][]*DynamicRootNode
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
	for nodeId, _ := range s.allNodesMap {
		report, err := s.getNodeAllStreams(nodeId)
		if err != nil || report == nil {
			continue
		}
		nodeStreamsMap[nodeId] = report
	}
	s.nodeStremasMap = nodeStreamsMap
}

/*
	for _, player := range stream.Players {
		for _, ip := range player.Ips {

			if _, ok := nodeMap[areaIspKey]; !ok {
				nodeMap[areaIspKey] = make([]*model.RtNode, 0)
			}
			nodeMap[areaIspKey] = append(nodeMap[areaIspKey], node)
		}
	}
*/
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
	for areaIsp, nodes := range s.rootNodesMap {
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

func (s *Parser) dump(nodeMap map[string][]*model.RtNode) {
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
}

func (s *Parser) buildRootNodesMap() {
	dynamicRootNodesMap, err := GetDynamicRootNodes(s.redisCli)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("map len", len(dynamicRootNodesMap))
	s.rootNodesMap = dynamicRootNodesMap
}

func (s *Parser) Run(sid, bkt string) {
	//nodeMap := s.getStreamNodes(sid, bkt)
	//s.dump(nodeMap)
	s.buildBucketStreamsInfo()
}

type NodeDetail struct {
	OnlineNum uint32
	RelayBw   float64
	Bw        float64
	MaxBw     float64
	Ratio     float64
	RelayType uint32
	Protocol  string
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

// TODO: 节点的每个网卡的带宽利用率, 这里的出口带宽不能使用streamreport上报的，其他业务线，
// 或者别的bucket下的流也会使用这个出口带宽
func (s *Parser) getNodeDetailMap(streamDetailMap map[string]map[string]map[string]StreamInfo,
	stream *model.StreamInfoRT, node *model.RtNode) map[string]map[string]*NodeDetail {
	nodeDetailMap := make(map[string]map[string]*NodeDetail) // key1:isp key2: area
	for _, player := range stream.Players {
		for _, ipInfo := range player.Ips {
			area, isp, err := getIpAreaIsp(s.ipParser, ipInfo.Ip)
			if err != nil {
				log.Println("getIpAreaIsp err", ipInfo.Ip, err)
				continue
			}
			if _, ok := nodeDetailMap[isp]; !ok {
				nodeDetailMap[isp] = make(map[string]*NodeDetail)
			}
			detail, ok := nodeDetailMap[isp][area]
			if !ok {
				nodeDetailMap[isp][area] = &NodeDetail{
					RelayType: stream.RelayType,
					Protocol:  player.Protocol,
					RelayBw:   float64(stream.RelayBandwidth), // TODO: 需要做转换
					OnlineNum: ipInfo.OnlineNum,
					Bw:        float64(ipInfo.Bandwidth), // TODO: 需要做转换
				}
			}
			detail.OnlineNum += ipInfo.OnlineNum
			detail.Bw += float64(ipInfo.Bandwidth)
		}
	}
	return nodeDetailMap
}

func (s *Parser) check(node *model.RtNode, streamInfo *model.NodeStreamInfo) bool {
	if !node.IsDynamic {
		return false
	}
	if streamInfo == nil {
		log.Println(node.Id, "not found stream info")
		return false
	}
	if time.Now().Unix()-streamInfo.LastUpdateTime > 300 {
		log.Println(node.Id, "stream offline",
			time.Unix(streamInfo.LastUpdateTime, 0).
				Format("2006-01-02 15:04:05 -0700 MST"))
		return false
	}
	if streamInfo.NodeId != node.Id {
		log.Println("check stream info node id err")
	}
	return true
}

/*
流id - ISP - 大区
map[string]map[string]map[string]StreamInfo
*/
func (s *Parser) buildBucketStreamsInfo(bkt string) {
	streamDetailMap := make(map[string]map[string]map[string]StreamInfo)
	for _, node := range s.allNodesMap {
		streamInfo := s.nodeStremasMap[node.Id]
		if !s.check(node, streamInfo) {
			continue
		}
		for _, stream := range streamInfo.Streams {
			if stream.Bucket != bkt {
				continue
			}
			s.getNodeDetailMap(streamDetailMap, stream, node)
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	sid := flag.String("sid", "", "流ID")
	bkt := flag.String("bkt", "douyu", "bucket ID")
	flag.Parse()
	if *sid == "" || *bkt == "" {
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
	parser.Run(*sid, *bkt)

}

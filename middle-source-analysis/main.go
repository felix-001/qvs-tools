package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	monitorUtil "github.com/qbox/mikud-live/cmd/monitor/common/util"
	"github.com/qbox/mikud-live/cmd/sched/common/consts"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/cmd/sched/dal"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	qlog "github.com/qbox/pili/base/qiniu/log.v1"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
	"github.com/redis/go-redis/v9"
	zlog "github.com/rs/zerolog/log"
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
	allNodeInfoMap           map[string]*NodeInfo
	streamDetailMap          map[string]map[string]map[string]*StreamInfo
	needCheckNode            bool
	file                     *os.File
}

func newParser(conf *Config, checkNode bool) *Parser {
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
	return &Parser{redisCli: redisCli, ipParser: ipParser, needCheckNode: checkNode}
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

func splitString(s string) (string, string) {
	// 从左到右遍历，找到最后一个数字的位置
	var lastDigitIndex int
	for i, char := range s {
		if !unicode.IsDigit(char) {
			lastDigitIndex = i
			break
		}
	}

	// 根据最后一个数字的位置分割字符串
	part1, part2 := s[:lastDigitIndex], s[lastDigitIndex:]

	return part1, part2
}

func (s *Parser) getStreamSourceNodeMap() {

}

var hdr = "流ID, 运营商, 大区, 在线人数, 边缘节点个数, ROOT节点个数, 放大比, 边缘节点详情, ROOT节点详情\n"
var streamRatioHdr = "流ID, 在线人数, 边缘节点个数, ROOT节点个数, 放大比\n"

func (s *Parser) dump() {
	csv := hdr
	cnt := 0
	var totalBw float64
	var totalRelayBw float64
	streamRatioMap := make(map[string]float64)
	streamRatioCsv := streamRatioHdr
	//rooms := make([]string, 0)
	roomMap := make(map[string][]string)
	roomOnlineMap := make(map[string]int)
	for streamId, streamDetail := range s.streamDetailMap {
		roomId, id := splitString(streamId)
		roomMap[roomId] = append(roomMap[roomId], id)
		cnt++
		var streamTotalBw float64
		var streamTotalRelayBw float64
		var streamTotalOnlineNum int
		var streamTotalEdgeNodeCount int
		var streamTotalRootNodeCount int
		for isp, detail := range streamDetail {
			for area, streamInfo := range detail {
				ratio := streamInfo.Bw / streamInfo.RelayBw
				csv += fmt.Sprintf("%s, %s, %s, %d, %d, %d, %.1f, %+v, %+v\n", streamId, isp, area,
					streamInfo.OnlineNum, len(streamInfo.EdgeNodes),
					len(streamInfo.RootNodes), ratio, streamInfo.EdgeNodes,
					streamInfo.RootNodes)
				totalBw += streamInfo.Bw
				totalRelayBw += streamInfo.RelayBw
				streamTotalBw += streamInfo.Bw
				streamTotalRelayBw += streamInfo.RelayBw
				streamTotalOnlineNum += int(streamInfo.OnlineNum)
				streamTotalEdgeNodeCount += len(streamInfo.EdgeNodes)
				streamTotalRootNodeCount += len(streamInfo.RootNodes)
				roomOnlineMap[roomId] += int(streamInfo.OnlineNum)
			}
		}
		streamRatioMap[streamId] = streamTotalBw / streamTotalRelayBw
		streamRatioCsv += fmt.Sprintf("%s, %d, %d, %d, %.1f\n", streamId,
			streamTotalOnlineNum, streamTotalEdgeNodeCount,
			streamTotalRootNodeCount, streamTotalBw/streamTotalRelayBw)
	}
	file := fmt.Sprintf("%d.csv", time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
	file = fmt.Sprintf("streams-%d.csv", time.Now().Unix())
	err = ioutil.WriteFile(file, []byte(streamRatioCsv), 0644)
	if err != nil {
		log.Println(err)
	}
	/*
		log.Println("cnt:", cnt)
		log.Printf("totalBw: %.1f, totalRelayBw: %.1f, totalRatio: %.1f", totalBw,
			totalRelayBw, totalBw/totalRelayBw)
		log.Println("room count:", len(roomMap))
		for roomId, ids := range roomMap {
			fmt.Println(roomId, ids)
		}
		log.Println("room - onlineNum info")
		for roomId, onlineNum := range roomOnlineMap {
			fmt.Println(roomId, onlineNum)
		}
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
	EdgeNodes []string
	RootNodes []string
	RelayBw   float64
	Bw        float64
	OnlineNum uint32
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

func (s *Parser) getNodeStreamDetail(stream *model.StreamInfoRT) (int, float64) {
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
	if s.needCheckNode && !s.checkNode(node) {
		log.Println("check node status err", node.Id)
		return false
	}
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

func (s *Parser) checkNode(node *model.RtNode) bool {
	if node == nil || !node.IsDynamic || node.RuntimeStatus != model.StateServing {
		return false
	}

	// 检查节点能力：状态、ability、service
	if !util.CheckNodeUsable(zlog.Logger, node, consts.TypeLive) {
		log.Printf("checkNode nodeId:%s, machineId:%s check not pass, type: %s\n",
			node.Id, node.MachineId, node.ResourceType)
		return false
	}

	if node.StreamdPorts.Http <= 0 || node.StreamdPorts.Https <= 0 || node.StreamdPorts.Wt <= 0 {
		log.Printf("getAllDynamicNodesReport check http,https,wt port failed, nodeId:%s\n", node.Id)
		return false
	}
	return true
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
			/*
				if stream.RelayType != 0 {
					log.Println("stream", stream.Key, "node", node.Id, "relaytype:", stream.RelayType)
				}
			*/
			if _, ok := s.streamDetailMap[stream.Key]; !ok {
				s.streamDetailMap[stream.Key] = make(map[string]map[string]*StreamInfo)
			}
			if _, ok := s.streamDetailMap[stream.Key][isp]; !ok {
				s.streamDetailMap[stream.Key][isp] = make(map[string]*StreamInfo)
			}
			onlineNum, bw := s.getNodeStreamDetail(stream)
			isRoot := s.isRoot(node)
			if streamInfo, ok := s.streamDetailMap[stream.Key][isp][area]; !ok {
				s.streamDetailMap[stream.Key][isp][area] = &StreamInfo{
					OnlineNum: uint32(onlineNum),
					Bw:        bw,
					//RelayBw:   convertMbps(stream.RelayBandwidth),
				}
				streamInfo := s.streamDetailMap[stream.Key][isp][area]
				if !isRoot {
					streamInfo.EdgeNodes = append(streamInfo.EdgeNodes, node.Id)
				} else {
					streamInfo.RootNodes = append(streamInfo.RootNodes, node.Id)
				}
				if stream.RelayType == 2 {
					streamInfo.RelayBw = convertMbps(stream.RelayBandwidth)
				}
			} else {
				streamInfo.OnlineNum += uint32(onlineNum)
				streamInfo.Bw += bw
				if stream.RelayType == 2 {
					streamInfo.RelayBw += convertMbps(stream.RelayBandwidth)
				}
				if !isRoot {
					streamInfo.EdgeNodes = append(streamInfo.EdgeNodes, node.Id)
				} else {
					streamInfo.RootNodes = append(streamInfo.RootNodes, node.Id)
				}
			}
		}
	}
	log.Println("total:", len(s.streamDetailMap))
}

func (s *Parser) dumpNodeStreams(node string) {
	for _, stream := range s.nodeStremasMap[node].Streams {
		fmt.Println("bucket:", stream.AppName, "stream:", stream.Key)
		for _, player := range stream.Players {
			fmt.Printf("\t%s\n", player.Protocol)
			for _, ipInfo := range player.Ips {
				fmt.Printf("\t\t ip: %s, onlineNum: %d, bw: %d\n", ipInfo.Ip, ipInfo.OnlineNum, ipInfo.Bandwidth)
			}
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	bkt := flag.String("bkt", "douyu", "bucket ID")
	node := flag.String("node", "", "node ID")
	monitor := flag.Bool("monitor", false, "node monitor")
	checkNode := flag.Bool("chknode", false, "是否需要检查节点的状态")
	flag.Parse()
	if *bkt == "" {
		flag.PrintDefaults()
		return
	}
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	qlog.SetOutputLevel(5)
	parser := newParser(&conf, *checkNode)
	if *monitor {
		parser.nodeMonitor()
		return
	}
	parser.buildAllNodesMap()
	parser.buildNodeStreamsMap()
	parser.buildRootNodesMap()
	if *node != "" {
		parser.dumpNodeStreams(*node)
		return
	}
	parser.buildBucketStreamsInfo(*bkt)
	parser.dump()
	parser.nodesNetinterfaceStatistics()
}

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"time"

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
	ck := newCk(conf)
	return &Parser{redisCli: redisCli, ipParser: ipParser, needCheckNode: checkNode, ck: ck}
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

func (s *Parser) dumpStreams(streamInfo *model.NodeStreamInfo) {
	for _, stream := range streamInfo.Streams {
		log.Printf("%+v", *stream)
	}
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	bkt := flag.String("bkt", "douyu", "bucket ID")
	node := flag.String("node", "", "node ID")
	stream := flag.String("stream", "", "stream ID")
	checkNode := flag.Bool("chknode", false, "是否需要检查节点的状态")
	monitor := flag.Bool("monitor", false, "node monitor")
	qlog.SetOutputLevel(5)
	flag.Parse()
	if *bkt == "" {
		flag.PrintDefaults()
		return
	}
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
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
	if *stream != "" {
		parser.getNodeUnavailableDetail("e7feb466-2b94-37d7-82d4-7f1002f6beb0-niulink64-site",
			"./node_info/nodeinfo-20240716094447.json")
		parser.dumpStreamDetail(*bkt, *stream)
		return
	}
	parser.dump()
	parser.dumpStreamsDetail(*bkt)
}

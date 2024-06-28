package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

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
	redisCli *redis.ClusterClient
	ipParser *ipdb.City
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

func (s *Parser) getStreamNodes(sid, bkt string) map[string][]*model.RtNode {
	allNodes, err := dal.GetAllNode(s.redisCli)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("nodes count", len(allNodes))
	nodeMap := make(map[string][]*model.RtNode)
	for _, node := range allNodes {
		report, err := s.getNodeAllStreams(node.Id)
		if err != nil || report == nil {
			continue
		}
		for _, stream := range report.Streams {
			if stream.Bucket != bkt || stream.Key != sid || stream.RelayBandwidth <= 0 {
				continue
			}
			for _, player := range stream.Players {
				for _, ip := range player.Ips {
					locate, err := publicUtil.GetLocate(ip.Ip, s.ipParser)
					if err != nil || locate == nil {
						log.Println("get locate err", err)
						continue
					}
					areaIspKey, _ := util.GetAreaIspKey(locate)
					if _, ok := nodeMap[areaIspKey]; !ok {
						nodeMap[areaIspKey] = make([]*model.RtNode, 0)
					}
					nodeMap[areaIspKey] = append(nodeMap[areaIspKey], node)
				}
			}
		}
	}
	return nodeMap
}

func (s *Parser) dump(nodeMap map[string][]*model.RtNode) {
	log.Println("nodesMap len:", len(nodeMap))
	for areaIsp, nodes := range nodeMap {
		log.Println(areaIsp, ":", len(nodes))
		fmt.Printf("\t")
		for _, node := range nodes {
			fmt.Printf("%s ", node.Id)
		}
	}
}

func (s *Parser) Run(sid, bkt string) {
	nodeMap := s.getStreamNodes(sid, bkt)
	s.dump(nodeMap)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	sid := flag.String("sid", "", "流ID")
	bkt := flag.String("bkt", "", "bucket ID")
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
	parser.Run(*sid, *bkt)

}

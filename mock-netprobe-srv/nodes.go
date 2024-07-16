package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	qconfig "github.com/qiniu/x/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/exp/rand"
)

func (s *NetprobeSrv) NodeChk() {
	var nodes []*model.RtNode
	if err := qconfig.LoadFile(&nodes, s.conf.NodesDataFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	cnt := 0
	availableNodes := 0
	lowTime0Cnt := 0
	aggNodeTime0Cnt := 0
	aggNodeTimeNot0Cnt := 0
	aggNodeNotUsableCnt := 0
	for _, node := range nodes {
		if node.ResourceType == "dedicated" {
			if !util.CheckNodeUsable(zerolog.Logger{}, node, "live") {
				log.Println("dedicated node", node.Id, "check usable fail")
				cnt++

			} else {
				lowTime := false
				for _, ip := range node.Ips {
					if ip.IPStreamProbe.LowThresholdTime == 0 {
						lowTime = true
					}
				}
				availableNodes++
				if lowTime {
					lowTime0Cnt++
				}
			}
		} else {
			if !util.CheckNodeUsable(zerolog.Logger{}, node, "live") {
				aggNodeNotUsableCnt++
			} else {
				log.Println("dedicated node", node.Id, "check usable fail")
				cnt++

				lowTime := false
				aggNodeTimeNot0 := false
				for _, ip := range node.Ips {
					if ip.IPStreamProbe.LowThresholdTime == 0 {
						lowTime = true
					} else if ip.IPStreamProbe.LowThresholdTime != 0 {
						aggNodeTimeNot0 = true
					}
				}
				if lowTime {
					aggNodeTime0Cnt++
				}
				if aggNodeTimeNot0 {
					aggNodeTimeNot0Cnt++
				}
			}
		}
	}
	log.Println("total", cnt, "availableNodes", availableNodes, "lowtime0Cnt", lowTime0Cnt, "aggNodeTime0Cnt", aggNodeTime0Cnt,
		"aggNodeTimeNot0Cnt", aggNodeTimeNot0Cnt, "aggNodeNotUsableCnt:", aggNodeNotUsableCnt)
}

func (s *NetprobeSrv) GeneOfflineData(paramMap map[string]string) string {
	area := paramMap["area"]
	offlineCntMap := map[string]int{}
	pipe := s.redisCli.Pipeline()
	idx := 0
	for _, node := range s.nodes {
		for _, ip := range node.Ips {
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			if node.ResourceType != "dedicated" {
				continue
			}
			areaIsp := s.getIpAreaIsp(ip.Ip)
			if areaIsp == area {
				rand.Seed(uint64(time.Now().UnixNano()))
				cnt := rand.Intn(100)
				if idx == 2 {
					log.Println("node", node.Id, "0")
					cnt = 0
				}
				offlineCntMap[node.Id] = cnt
				for i := 0; i < cnt; i++ {
					//log.Println("i", i)
					_, err := pipe.ZAdd(context.Background(), "dynamic_node_offline_cnt_"+node.Id, redis.Z{
						Member: time.Now().Unix() - int64(i),
						Score:  float64(time.Now().Unix() - int64(i)),
					}).Result()
					if err != nil {
						log.Println(err)
					}
				}
				idx++
			}
		}
	}
	_, err := pipe.Exec(context.Background())
	if err != nil {
		log.Println("pipe exec err", err)
	}
	jsonbody, err := json.Marshal(offlineCntMap)
	if err != nil {
		log.Println(err)
	}
	return string(jsonbody)
}

func (s *NetprobeSrv) NodeInfo(paramMap map[string]string) string {
	nodeId := paramMap["node"]
	log.Println(len(s.nodes))
	for _, node := range s.nodes {
		if node.Id == nodeId {
			log.Println("found the node")
			//node.RuntimeStatus = "Offline"
			fmt.Printf("%+v\n", node)
			jsonbody, err := json.Marshal(node)
			if err != nil {
				log.Println(err)
			}
			return string(jsonbody)
		}
	}
	log.Println("node not found:", nodeId)
	return "fail"
}

func (s *NetprobeSrv) SetLowThresholdTime(paramMap map[string]string) string {
	t := paramMap["time"]
	node := paramMap["node"]
	num, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return fmt.Sprintf("parse int err, %v", err)
	}
	if extra, ok := s.nodeExtras[node]; ok {
		extra.LowThresholdTime = num
	} else {
		s.nodeExtras[node] = &NodeExtra{LowThresholdTime: num}
	}
	return "success"
}

func (s *NetprobeSrv) SetNodeRuntimeState(paramMap map[string]string) string {
	nodeId := paramMap["node"]
	state := paramMap["state"]
	log.Println(nodeId, state)
	for i, node := range s.nodes {
		if node.Id == nodeId {
			s.nodes[i].RuntimeStatus = state
		}
	}
	return "success"
}

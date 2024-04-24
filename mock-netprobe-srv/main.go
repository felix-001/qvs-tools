package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/qbox/mikud-live/cmd/sched/common/util"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
	"github.com/redis/go-redis/v9"
	zerolog "github.com/rs/zerolog"
)

const (
	confFile = "/usr/local/etc/mock-netprobe-srv.conf"
)

type Config struct {
	RedisAddrs    []string    `json:"redis_addrs"`
	NodesDataFile string      `json:"nodes_data_file"`
	IPDB          ipdb.Config `json:"ipdb"`
}

type NetprobeSrv struct {
	redisCli *redis.ClusterClient
	conf     Config
	nodes    []*model.RtNode
	ipParser *ipdb.City
}

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

func (s *NetprobeSrv) Load() {
	if err := qconfig.LoadFile(&s.nodes, s.conf.NodesDataFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	log.Println("node count:", len(s.nodes))
}

func (s *NetprobeSrv) Run() {
	for range time.Tick(time.Duration(10) * time.Second) {
		log.Println("update nodes, count:", len(s.nodes))
		for _, node := range s.nodes {
			for i := range node.Ips {
				node.Ips[i].IPStreamProbe.LowThresholdTime = time.Now().Unix()
			}
			if node.Id == "bf81488f-053b-3e70-b0a0-4aae62203e62-niulink64-site" {
				log.Println(node.RuntimeStatus)
			}
			bytes, err := json.Marshal(node)
			if err != nil {
				log.Println(err)
				return
			}
			_, err = s.redisCli.HSet(context.Background(), model.NetprobeRtNodesMap, node.Id, bytes).Result()
			if err != nil {
				log.Printf("write node info to redis err, %+v\n", err)
			}
		}
	}
}

func (s *NetprobeSrv) NodeFreeze(nodeId string) {
	log.Println(len(s.nodes))
	for i, node := range s.nodes {
		if node.Id == nodeId {
			log.Println("found the node")
			s.nodes[i].RuntimeStatus = "Offline"
			//node.RuntimeStatus = "Offline"
			return
		}
	}
	log.Println("node not found:", nodeId)
}

func (s *NetprobeSrv) NodeInfo(nodeId string) *model.RtNode {
	log.Println(len(s.nodes))
	for _, node := range s.nodes {
		if node.Id == nodeId {
			log.Println("found the node")
			//node.RuntimeStatus = "Offline"
			fmt.Printf("%+v\n", node)
			return node
		}
	}
	log.Println("node not found:", nodeId)
	return nil
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
			out += fmt.Sprintf("node: %s inMpbs: %.0f maxInMbps: %.0f inRatio: %.1f outMbps: %.0f maxOutMbps: %.0f outRatio: %.1f\n",
				node.Id, inBw, maxInBw, inBw/maxInBw, outBw, maxOutBw, outBw/maxOutBw)
			nodeCnt++
		}
	}
	out += fmt.Sprintf("node count: %d\n", nodeCnt)

	return out
}

func (s *NetprobeSrv) GetAreaIspNodesInfo(needAreaIsp string) []*model.RtNode {
	areaIspGroup := make(map[string][]*model.RtNode)
	for _, node := range s.nodes {
		for _, ip := range node.Ips {
			locate, err := s.ipParser.Find(ip.Ip)
			if err != nil {
				log.Println("get locate of ip", ip.Ip, "err", err)
				continue
			}
			areaIpsKey, _ := util.GetAreaIspKey(locate)
			areaIsp := strings.TrimPrefix(areaIpsKey, util.AreaIspKeyPrefix)
			areaIspGroup[areaIsp] = append(areaIspGroup[areaIsp], node)
			break
		}
	}
	log.Println("node count:", len(areaIspGroup[needAreaIsp]))
	return areaIspGroup[needAreaIsp]
}

func (s *NetprobeSrv) findNode(nodeId string) *model.RtNode {
	for _, node := range s.nodes {
		if node.Id == nodeId {
			return node
		}
	}
	return nil
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
				//s.nodes[i].Ips[j].InMBps =
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
				s.nodes[i].Ips[j].InMBps = ip.MaxInMBps - 1000
				//s.nodes[i].Ips[j].InMBps =
			}
			break
		}
	}
	return "success"
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	nodeChk := flag.Bool("chk", false, "检查节点")
	flag.Parse()
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	log.Println(conf)
	ipParser, err := ipdb.NewCity(conf.IPDB)
	if err != nil {
		log.Fatalf("[IPDB NewCity] err: %+v\n", err)
	}
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})
	err = redisCli.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	app := NetprobeSrv{redisCli: redisCli, conf: conf, ipParser: ipParser}
	app.Load()
	if *nodeChk {
		app.NodeChk()
		return
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		app.NodeFreeze(nodeId)
		fmt.Fprintf(w, "success")
	}
	nodeInfoHandler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		node := app.NodeInfo(nodeId)
		jsonbody, err := json.Marshal(node)
		if err != nil {
			log.Println(err)
		}
		fmt.Fprintf(w, string(jsonbody))
	}
	costBwHandler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		app.NodeFreeze(nodeId)
		fmt.Fprintf(w, "success")
	}
	getAreaIspNodesHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["areaIsp"]
		nodes := app.GetAreaIspNodesInfo(areaIsp)
		jsonbody, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Println(err)
		}
		fmt.Fprintf(w, string(jsonbody))
	}
	getAreaIspRootBwInfoHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["areaIsp"]
		info := app.GetAreaIspRootBwInfo(areaIsp)
		fmt.Fprintf(w, info)
	}

	clearBwHandler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		app.ClearBw(nodeId)
		fmt.Fprintf(w, "success")
	}

	fillInBwHandler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		app.FillInBw(nodeId)
		fmt.Fprintf(w, "success")
	}

	fillOutBwHandler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		app.FillOutBw(nodeId)
		fmt.Fprintf(w, "success")
	}

	go func() {
		router := mux.NewRouter()
		router.HandleFunc("/freeze/{id}", handler)
		router.HandleFunc("/node/{id}", nodeInfoHandler)
		router.HandleFunc("/costbw/{id}", costBwHandler)
		router.HandleFunc("/nodes/{areaIsp}", getAreaIspNodesHandler)
		router.HandleFunc("/area/{areaIsp}/rootBwInfo", getAreaIspRootBwInfoHandler)
		router.HandleFunc("/node/{id}/clearbw", clearBwHandler)
		router.HandleFunc("/node/{id}/fillInbw", fillInBwHandler)
		router.HandleFunc("/node/{id}/fillOutbw", fillOutBwHandler)
		http.Handle("/", router)
		http.ListenAndServe(":9090", nil)
	}()
	app.Run()
}

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
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

type NodeExtra struct {
	StreamInfo       model.NodeStreamInfo
	LowThresholdTime int64
}

type Config struct {
	RedisAddrs    []string    `json:"redis_addrs"`
	NodesDataFile string      `json:"nodes_data_file"`
	IPDB          ipdb.Config `json:"ipdb"`
}

type NetprobeSrv struct {
	redisCli   *redis.ClusterClient
	conf       Config
	nodes      []*model.RtNode
	ipParser   *ipdb.City
	nodeExtras map[string]*NodeExtra
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
				node.Ips[i].IPStreamProbe.LowThresholdTime = time.Now().Unix() - 3*3600
				if extra, ok := s.nodeExtras[node.Id]; ok {
					log.Println("found extra", node.Id, extra.LowThresholdTime)
					node.Ips[i].IPStreamProbe.LowThresholdTime = extra.LowThresholdTime
				}
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
			/*
				nodeStreamInfo := model.NodeStreamInfo{
					Streams: []*model.StreamInfoRT{
						{
							StreamName: "test",
						},
					},
					NodeId:         node.Id,
					LastUpdateTime: time.Now().Unix(),
				}
				data, err := json.Marshal(nodeStreamInfo)
				if err != nil {
					log.Println(err)
					continue
				}
				_, err = s.redisCli.Set(context.Background(), "stream_report_"+node.Id, string(data), time.Hour*24*30).Result()
				if err != nil {
					log.Println(err)
				}
			*/
		}
	}
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

func (s *NetprobeSrv) DumpAreaIsp(paramMap map[string]string) string {
	areaIspMap := map[string]int{}
	for _, node := range s.nodes {
		for _, ip := range node.Ips {
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			areaIsp := s.getIpAreaIsp(ip.Ip)
			areaIspMap[areaIsp] += 1
			break
		}
	}
	jsonbody, err := json.Marshal(areaIspMap)
	if err != nil {
		log.Println(err)
	}
	return string(jsonbody)
}

func (s *NetprobeSrv) StreamReport(paramMap map[string]string) string {
	node := paramMap["node"]
	body := paramMap["body"]
	stream := paramMap["stream"]
	bucket := paramMap["bucket"]

	ipOnlineNumMap := map[string]int{}
	if err := json.Unmarshal([]byte(body), &ipOnlineNumMap); err != nil {
		return fmt.Sprintf("unmashal err, %v", err)
	}

	var ips []*model.IpInfo
	for ip, onlineNum := range ipOnlineNumMap {
		ipInfo := &model.IpInfo{
			Ip:        ip,
			OnlineNum: uint32(onlineNum),
		}
		ips = append(ips, ipInfo)
	}
	nodeStreamInfo := model.NodeStreamInfo{
		NodeId:         node,
		LastUpdateTime: time.Now().Unix(),
		Streams: []*model.StreamInfoRT{
			{
				AppName:    bucket,
				Bucket:     bucket,
				Key:        stream,
				StreamName: stream,
				Players: []*model.PlayerInfo{
					{
						Ips: ips,
					},
				},
			},
		},
	}

	bytes, err := json.Marshal(&nodeStreamInfo)
	if err != nil {
		return fmt.Sprintf("marshal err, %v", err)
	}

	_, err = s.redisCli.Set(context.Background(), "stream_report_"+node, bytes, time.Hour*24*30).Result()
	if err != nil {
		log.Println(err)
		return fmt.Sprintf("redis err, %v", err)
	}

	return "success"
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
				rand.Seed(time.Now().UnixNano())
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

func (s *NetprobeSrv) getIpAreaIsp(ip string) string {
	locate, err := s.ipParser.Find(ip)
	if err != nil {
		log.Println("get locate of ip", ip, "err", err)
		return ""
	}
	areaIpsKey, _ := util.GetAreaIspKey(locate)
	areaIsp := strings.TrimPrefix(areaIpsKey, util.AreaIspKeyPrefix)
	return areaIsp
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

func (s *NetprobeSrv) GetAreaInfo(areaIsp string) string {
	info := map[string]float64{}
	for _, node := range s.nodes {
		for _, ip := range node.Ips {
			if ip.IsIPv6 {
				continue
			}
			if publicUtil.IsPrivateIP(ip.Ip) {
				continue
			}
			locate, err := s.ipParser.Find(ip.Ip)
			if err != nil {
				log.Println("get locate of ip", ip.Ip, "err", err)
				continue
			}
			areaIpsKey, _ := util.GetAreaIspKey(locate)
			areaIsp_ := strings.TrimPrefix(areaIpsKey, util.AreaIspKeyPrefix)
			if areaIsp != areaIsp_ {
				continue
			}
			info[node.Id] += ip.MaxOutMBps
		}
	}
	pairs := SortFloatMap(info)
	jsonbody, err := json.Marshal(pairs)
	if err != nil {
		log.Println(err)
	}
	return string(jsonbody)
}

type Pair struct {
	Key string
	Val float64
}

func SortFloatMap(m map[string]float64) []Pair {
	pairs := []Pair{}
	for k, v := range m {
		pairs = append(pairs, Pair{Key: k, Val: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Val < pairs[j].Val
	})
	return pairs
}

func (s *NetprobeSrv) Demo(paramMap map[string]string) string {
	return "demo running " + paramMap["foo"] + " " + paramMap["test"]
}

type Router struct {
	Path    string
	Params  []string
	Handler func(paramMap map[string]string) string
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
	app := NetprobeSrv{redisCli: redisCli, conf: conf, ipParser: ipParser, nodeExtras: make(map[string]*NodeExtra)}
	app.Load()
	if *nodeChk {
		app.NodeChk()
		return
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

	getAreaInfoHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["areaIsp"]
		info := app.GetAreaInfo(areaIsp)
		fmt.Fprintf(w, info)
	}

	clearBwHandler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		app.ClearBw(nodeId)
		fmt.Fprintf(w, "success")
	}

	fillAreaBwHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["area"]
		app.FillAreaBw(areaIsp)
		fmt.Fprintf(w, "success")
	}
	fillIspBwHandler := func(w http.ResponseWriter, req *http.Request) {
		isp := mux.Vars(req)["isp"]
		app.FillIspBw(isp)
		fmt.Fprintf(w, "success")
	}

	routers := []Router{
		{
			"/demo",
			[]string{"foo", "test"},
			app.Demo,
		},
		{
			"/fillbw",
			[]string{"node", "type"},
			app.FillBw,
		},
		{
			"/runtimeState",
			[]string{"node", "state"},
			app.SetNodeRuntimeState,
		},
		{
			"/nodeinfo",
			[]string{"node"},
			app.NodeInfo,
		},
		{
			"/dumpAreaIsp",
			[]string{""},
			app.DumpAreaIsp,
		},
		{
			"/genneOfflineData",
			[]string{"area"},
			app.GeneOfflineData,
		},
		{
			"/streamreport",
			[]string{"node", "bucket", "stream"},
			app.StreamReport,
		},
		{
			"/lowThresholdTime",
			[]string{"node", "time"},
			app.SetLowThresholdTime,
		},
	}

	go func() {
		router := mux.NewRouter()
		router.HandleFunc("/nodes/{areaIsp}", getAreaIspNodesHandler)
		router.HandleFunc("/area/{areaIsp}/rootBwInfo", getAreaIspRootBwInfoHandler)
		router.HandleFunc("/node/{id}/clearbw", clearBwHandler)
		router.HandleFunc("/area/{area}/fillAreaBw", fillAreaBwHandler)
		router.HandleFunc("/isp/{isp}/fillIspBw", fillIspBwHandler)
		router.HandleFunc("/area/{areaIsp}", getAreaInfoHandler)
		for _, r := range routers {
			commonHandler := func(w http.ResponseWriter, req *http.Request) {
				log.Println(req)
				var handler func(paramMap map[string]string) string
				var params *[]string
				for _, r := range routers {
					if r.Path == req.URL.Path {
						log.Println(r.Path)
						handler = r.Handler
						params = &r.Params
						break
					}
				}
				paramMap := map[string]string{}
				for _, param := range *params {
					val := req.URL.Query().Get(param)
					paramMap[param] = val
				}
				body, err := ioutil.ReadAll(req.Body)
				if err != nil {
					http.Error(w, "Error reading request body", http.StatusBadRequest)
					return
				}
				defer req.Body.Close()
				fmt.Println("Request Body:", string(body))
				paramMap["body"] = string(body)
				res := handler(paramMap)
				fmt.Fprintln(w, res)
			}
			router.HandleFunc(r.Path, commonHandler)
		}
		http.Handle("/", router)
		http.ListenAndServe(":9098", nil)
	}()
	app.Run()
}

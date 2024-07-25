package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
	"github.com/redis/go-redis/v9"
)

const (
	confFile = "/usr/local/etc/mock-netprobe-srv.conf"
)

func (s *NetprobeSrv) LoadNodes() {
	if err := qconfig.LoadFile(&s.nodes, s.conf.NodesDataFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	log.Println("node count:", len(s.nodes))
}

func newApp(conf Config) *NetprobeSrv {
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
	app := NetprobeSrv{
		redisCli:        redisCli,
		conf:            conf,
		ipParser:        ipParser,
		nodeExtras:      make(map[string]*NodeExtra),
		streamReportMap: make(map[string]map[string]map[string]map[string]int),
	}
	return &app
}

func main1() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	nodeChk := flag.Bool("chk", false, "检查节点")
	flag.Parse()
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	log.Println(conf)
	app := newApp(conf)
	app.LoadNodes()
	if *nodeChk {
		app.NodeChk()
		return
	}

	routers := app.routers()

	go func() {
		router := app.router()
		app.initRouter(routers, router)
		http.Handle("/", router)
		http.ListenAndServe(":9098", nil)
	}()
	app.Run()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	bkt := flag.String("bkt", "douyuflv", "bucket")
	node := flag.String("node", "02f67b4b-80d0-3016-b08d-cf6dbfee2ab3-niulink64-site", "node")
	stream := flag.String("stream", "stream01", "stream")
	onlineNum := flag.Int("num", 310, "online num")
	ip := flag.String("ip", "59.83.196.49", "ip")
	redisAddr := flag.String("redis", "10.20.54.24:8200", "redis")
	flag.Parse()

	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      []string{*redisAddr},
		MaxRetries: 3,
		PoolSize:   30,
	})
	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	streamName := *bkt + ":" + *stream
	for {
		data, err := redisCli.Get(context.Background(), "stream_report_"+*node).Result()
		if err != nil {
			log.Println(err)
			continue
		}
		nodeStreamInfo := model.NodeStreamInfo{}
		if err = json.Unmarshal([]byte(data), &nodeStreamInfo); err != nil {
			log.Println(err)
			continue
		}
		log.Println("streams count:", len(nodeStreamInfo.Streams))
		found := false
		for _, streamInfo := range nodeStreamInfo.Streams {
			if streamInfo.Key == *stream {
				found = true
				log.Println("found stream", *stream)
				break
			}
		}
		if found {
			time.Sleep(time.Second * 5)
			continue

		}
		log.Println("stream", *stream, "not found, update")
		streamInfo := &model.StreamInfoRT{
			StreamName: streamName,
			Key:        *stream,
			Bucket:     *bkt,
			Players: []*model.PlayerInfo{
				{
					Protocol: "flv",
					Ips: []*model.IpInfo{
						{Ip: *ip, OnlineNum: uint32(*onlineNum)},
					},
				},
			},
		}

		nodeStreamInfo.Streams = append(nodeStreamInfo.Streams, streamInfo)
		bytes, err := json.Marshal(nodeStreamInfo)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = redisCli.Set(context.Background(), "stream_report_"+*node, string(bytes), time.Second*30).Result()
		if err != nil {
			log.Println(err)
		}
		time.Sleep(time.Second * 5)
	}
}

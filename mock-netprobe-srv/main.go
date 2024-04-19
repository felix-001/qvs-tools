package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/qbox/mikud-live/common/model"
	qconfig "github.com/qiniu/x/config"
	"github.com/redis/go-redis/v9"
)

const (
	confFile = "/usr/local/etc/mock-netprobe-srv.conf"
)

type Config struct {
	RedisAddrs    []string `json:"redis_addrs"`
	NodesDataFile string   `json:"nodes_data_file"`
}

type NetprobeSrv struct {
	redisCli *redis.ClusterClient
	conf     Config
}

func (s *NetprobeSrv) Run() {
	var nodes []*model.RtNode
	if err := qconfig.LoadFile(&nodes, s.conf.NodesDataFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	log.Println("node count:", len(nodes))
	for range time.Tick(time.Duration(10) * time.Second) {
		for _, node := range nodes {
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var conf Config
	if err := qconfig.LoadFile(&conf, confFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	log.Println(conf)
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      conf.RedisAddrs,
		MaxRetries: 3,
		PoolSize:   30,
	})
	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	app := NetprobeSrv{redisCli: redisCli, conf: conf}
	app.Run()
}

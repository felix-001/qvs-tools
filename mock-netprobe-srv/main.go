package main

import (
	"context"
	"log"

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
}

func (s *NetprobeSrv) Run() {
	res, err := s.redisCli.Keys(context.Background(), "*").Result()
	log.Println(res, err)
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
	app := NetprobeSrv{redisCli: redisCli}
	app.Run()
}

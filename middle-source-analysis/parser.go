package main

import (
	"context"
	"log"

	qlog "github.com/qbox/pili/base/qiniu/log.v1"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

func newParser(conf *Config) *Parser {
	redisCli := &redis.ClusterClient{}
	if conf.Redis {
		redisCli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:      conf.RedisAddrs,
			MaxRetries: 3,
			PoolSize:   30,
		})

		err := redisCli.Ping(context.Background()).Err()
		if err != nil {
			log.Fatalf("%+v", err)
		}
	}
	var ipParser *ipdb.City
	if conf.NeedIpParer {
		qlog.SetOutputLevel(5)
		var err error
		ipParser, err = ipdb.NewCity(conf.IPDB)
		if err != nil {
			log.Fatalf("[IPDB NewCity] err: %+v\n", err)
		}
	}
	ck := newCk(conf)
	return &Parser{
		redisCli: redisCli,
		ipParser: ipParser,
		ck:       ck,
		conf:     conf,
	}
}

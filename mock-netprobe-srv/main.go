package main

import (
	"context"
	"flag"
	"log"
	"net/http"

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

func main() {
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

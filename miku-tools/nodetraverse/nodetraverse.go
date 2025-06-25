package nodetraverse

import (
	"context"
	"fmt"

	public "github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type NodeCallback interface {
	OnIp(node *public.RtNode, ip *public.RtIpStatus)
	OnNode(node *public.RtNode, ipParser *ipdb.City)
}

var modules = []NodeCallback{}

func Register(module NodeCallback) {
	modules = append(modules, module)
}

func GetMoudleCnt() int {
	return len(modules)
}

func Traverse(addrs []string, conf ipdb.Config) {
	fmt.Println("NodeTraverse")
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      addrs,
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	allNodes, err := public.GetAllRTNodes(log.Logger, redisCli)
	if err != nil {
		log.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
		return
	}

	ipParser, err := ipdb.NewCity(conf)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, node := range allNodes {
		for _, module := range modules {
			module.OnNode(node, ipParser)
		}
		for _, ip := range node.Ips {
			for _, module := range modules {
				module.OnIp(node, &ip)
			}
		}
	}
}

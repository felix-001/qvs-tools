package stream

import (
	"encoding/json"
	"fmt"
	"log"
	"middle-source-analysis/config"
	"middle-source-analysis/util"

	"github.com/qbox/mikud-live/cmd/sched/model"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

var (
	NodeStremasMap map[string]*model.NodeStreamInfo
	Conf           *config.Config
	IpParser       *ipdb.City
	RedisCli       *redis.ClusterClient
)

func GetNodesByStreamId() map[string]model.StreamNodeDetailList {

	addr := fmt.Sprintf("http://10.34.146.62:6060/api/v1/bucket/%s/stream/%s/nodes",
		Conf.Bucket, Conf.Stream)
	resp, err := util.Get(addr)
	if err != nil {
		return nil
	}
	var nodesMap map[string]model.StreamNodeDetailList
	if err := json.Unmarshal([]byte(resp), &nodesMap); err != nil {
		log.Println(err)
		return nil
	}
	return nodesMap
}

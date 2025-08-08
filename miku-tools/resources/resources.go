package resources

import (
	"mikutool/public/util"

	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	"github.com/qbox/mikud-live/common/dal/mongo"
	"github.com/qbox/mikud-live/common/repository/filter"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

type Resources struct {
	Redis         *redis.ClusterClient
	IpParser      *ipdb.City
	Ck            *util.Ck
	Mongo         *mongo.MongoClient
	NodeFilterCol filter.AvailableResourceRepository
	// key1: isp key2: prov value: ip
	V4Ips     map[string]map[string]string
	V6Ips     map[string]map[string]string
	DnsPodCli *tencent_dnspod.TencentClient
}

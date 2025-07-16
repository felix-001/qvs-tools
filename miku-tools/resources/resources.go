package resources

import (
	"mikutool/public/util"

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
}

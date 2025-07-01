package resources

import (
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

type Resources struct {
	Redis    *redis.ClusterClient
	IpParser *ipdb.City
}

package manager

import (
	"context"
	"flag"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/miku"
	"mikutool/miku/nodemgr"
	"mikutool/miku/streammgr"
	"mikutool/public/util"
	"mikutool/qvs/staging"
	"mikutool/resources"
	"reflect"
	"strings"

	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	publicCommon "github.com/qbox/mikud-live/common"
	"github.com/qbox/mikud-live/common/dal/mongo"
	"github.com/qbox/mikud-live/common/repository/filter"
	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type CommandManager struct {
	commands    map[string]*Command
	config      *config.Config
	resources   resources.Resources
	nodeMgr     *nodemgr.NodeMgr
	streamMgr   *streammgr.StreamMgr
	mikuStagMgr *staging.MikuStagMgr
	miku        *miku.Miku
}

func NewCommandManager() *CommandManager {
	return &CommandManager{
		commands:    make(map[string]*Command),
		nodeMgr:     nodemgr.NewNodeMgr(),
		streamMgr:   streammgr.NewStreamMgr(),
		mikuStagMgr: staging.NewMikuStagMgr(),
		miku:        miku.NewMiku(),
	}
}

func (m *CommandManager) Exec() {
	if m.config.H {
		if m.config.Cmd != "" {
			cmd, ok := m.commands[m.config.Cmd]
			if !ok {
				log.Println("command:", m.config.Cmd, "not found")
				return
			}
			fmt.Println(cmd.Desc)
			return
		}
		m.usage()
		return
	}
	if m.config.Help {
		m.usage()
		flag.PrintDefaults()
		return
	}
	cmd, ok := m.commands[m.config.Cmd]
	if !ok {
		log.Println("command:", m.config.Cmd, "not found")
		return
	}
	m.loadResources(cmd)
	cmd.Handler()
}

func (m *CommandManager) Init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	m.config = config.Load()
	m.config.ParseConsole()
	m.nodeMgr.SetConf(m.config)
	m.streamMgr.SetConf(m.config)
	m.miku.SetConf(m.config)
}

func (m *CommandManager) Register() {
	typ := reflect.TypeOf(m)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if strings.HasPrefix(method.Name, "Cmd") {
			key := strings.ToLower(strings.TrimPrefix(method.Name, "Cmd"))
			//log.Println("register command:", key)
			results := method.Func.Call([]reflect.Value{reflect.ValueOf(m)})
			if len(results) > 0 && results[0].Type().String() == "*manager.Command" {
				m.commands[key] = results[0].Interface().(*Command)
			}
		}
	}
}

func (m *CommandManager) loadResources(cmd *Command) {
	var err error

	if m.config.Local {
		if cmd.NeedNodeInfo {
			m.nodeMgr.LoadNodes()
		}
		return
	}
	m.resources.V4Ips = util.LoadV4Ips()
	m.resources.V6Ips = util.LoadV6Ips()
	if cmd.NeedIpParser {
		m.resources.IpParser, err = ipdb.NewCity(m.config.IPDB)
		if err != nil {
			log.Println("load ipdb err", err)
		}
	}
	if cmd.NeedRedis {
		m.resources.Redis = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:      m.config.RedisAddrs,
			MaxRetries: 3,
			PoolSize:   30,
		})
		err = m.resources.Redis.Ping(context.Background()).Err()
		if err != nil {
			log.Println(err)
		}
		log.Println("loadResources Ping redis success")
	}
	if cmd.NeedCK {
		m.resources.Ck = util.NewCk(m.config)
	}
	if cmd.NeedMongo {
		mongoClient, err := mongo.NewMongoClient(m.config.MongoConf)
		if err != nil {
			log.Printf("[NewMongoClient] err: %+v\n", err)
			return
		}
		m.resources.Mongo = mongoClient
		availableResourceCol := mongoClient.Collection(publicCommon.AvailableResourceCollection)
		m.resources.NodeFilterCol = filter.NewAvailableResourceRepo(availableResourceCol, zerolog.Logger{})
	}
	if cmd.NeedDnsPod {
		if m.config.DnsPod.SecretKey == "" {
			log.Fatal("dns pod secret key is empty\n")
			return
		}
		var err error
		m.resources.DnsPodCli, err = tencent_dnspod.NewTencentClient(m.config.DnsPod)
		if err != nil {
			log.Println("load dns pod client err", err)
		}
	}
	m.nodeMgr.SetResources(m.resources)
	m.miku.SetResources(m.resources)
	if cmd.NeedNodeInfo {
		m.nodeMgr.LoadNodes()
	}
}

func (m *CommandManager) usage() {
	for key, cmd := range m.commands {
		fmt.Printf("%s\n\t%s\n", key, cmd.Desc)
		fmt.Println()
	}
}

func (m *CommandManager) Locate() {
	info, err := m.resources.IpParser.Find(m.config.Ip)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(info)
}

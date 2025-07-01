package manager

import (
	"context"
	"log"
	"mikutool/config"
	"mikutool/resources"
	"reflect"
	"strings"

	"github.com/qbox/pili/common/ipdb.v1"
	"github.com/redis/go-redis/v9"
)

type CommandManager struct {
	commands  map[string]*Command
	config    *config.Config
	resources resources.Resources
}

func NewCommandManager() *CommandManager {
	return &CommandManager{
		commands: make(map[string]*Command),
	}
}

func (m *CommandManager) Exec() {
	cmd, ok := m.commands[m.config.Cmd]
	if !ok {
		log.Println("command:", m.config.Cmd, "not found")
		return
	}
	cmd.Handler()
}

func (m *CommandManager) Init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	m.config = config.Load()
	m.config.ParseConsole()
}

func (m *CommandManager) Register() {
	typ := reflect.TypeOf(m)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if strings.HasPrefix(method.Name, "Cmd") {
			key := strings.ToLower(strings.TrimPrefix(method.Name, "Cmd"))
			//log.Println("register command:", key)
			m.commands[key] = &Command{
				Handler: func() {
					method := typ.Method(i)
					method.Func.Call([]reflect.Value{reflect.ValueOf(m)})
				},
			}
		}
	}
}

func (m *CommandManager) loadResources(resources []string) {
	for _, resource := range resources {
		var err error
		switch resource {
		case ResourceIpParser:
			m.resources.IpParser, err = ipdb.NewCity(m.config.IPDB)
			if err != nil {
				log.Println("load ipdb err", err)
			}
		case ResourceRedis:
			m.resources.Redis = redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:      m.config.RedisAddrs,
				MaxRetries: 3,
				PoolSize:   30,
			})
			err = m.resources.Redis.Ping(context.Background()).Err()
			if err != nil {
				log.Println(err)
			}
		}
	}
}

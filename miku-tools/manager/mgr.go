package manager

import (
	"context"
	"flag"
	"fmt"
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
	if m.config.Help {
		m.usage()
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
	}
}

func (m *CommandManager) usage() {
	for key, cmd := range m.commands {
		fmt.Printf("%s\n\t%s\n", key, cmd.Desc)
		fmt.Println()
	}
	flag.PrintDefaults()
}

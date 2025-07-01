package manager

import (
	"log"
	"mikutool/config"
	"reflect"
	"strings"
)

type CommandManager struct {
	commands map[string]*Command
	config   *config.Config
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

package manager

import "log"

func (m *CommandManager) CmdList(cmd string) {
	if cmd == "cfg" {
		return
	}
	log.Println("list")
	m.loadResources([]string{ResourceIpParser, ResourceRedis})
}

func (m *CommandManager) CmdStreams() {
	log.Println("streams")
	m.loadResources([]string{ResourceIpParser, ResourceRedis})
}

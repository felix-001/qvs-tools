package manager

import "log"

func (m *CommandManager) CmdList() {
	log.Println("list")
	m.loadResources([]string{ResourceIpParser, ResourceRedis})
}

func (m *CommandManager) CmdStreams() {
	log.Println("streams")
	m.loadResources([]string{ResourceIpParser, ResourceRedis})
}

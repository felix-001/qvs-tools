package manager

import "log"

func (m *CommandManager) CmdStreams() *Command {
	handler := func() {
		log.Println("streams")
	}
	cmd := &Command{
		Desc:         "dump流粒度放大比",
		Handler:      handler,
		NeedIpParser: true,
		NeedRedis:    true,
	}
	return cmd
}

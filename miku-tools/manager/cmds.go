package manager

import "mikutool/public/util"

func (m *CommandManager) CmdHttp() *Command {
	handler := func() {
		util.Http(m.config)
	}
	cmd := &Command{
		Desc:    "qn http客户端",
		Handler: handler,
	}
	return cmd
}

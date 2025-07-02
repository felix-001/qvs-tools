package manager

import (
	"mikutool/miku"
	"mikutool/miku/users"
	"mikutool/public/util"
	"mikutool/qvs"
	"mikutool/qvs/mock"
)

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

func (m *CommandManager) CmdMock() *Command {
	handler := func() {
		mock.MockSrv()
	}
	cmd := &Command{
		Desc:    "mock tracker&themisd&server",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdDyPlay() *Command {
	handler := func() {
		users.DyPlay(m.config, m.nodeMgr)
	}
	cmd := &Command{
		Desc:    "播放dy xs流",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdAkSk() *Command {
	handler := func() {
		util.GetAkSk(m.config)
	}
	cmd := &Command{
		Desc:    "ak sk",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdPlayCheck() *Command {
	handler := func() {
		miku.Playcheck(m.config)
	}
	cmd := &Command{
		Desc:    "请求playcheck 302接口",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdInvite() *Command {
	handler := func() {
		qvs.Invite(m.config)
	}
	cmd := &Command{
		Desc:    "请求invite",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdKodo() *Command {
	handler := func() {
		util.SignResource(m.config)
	}
	cmd := &Command{
		Desc:    "kodo",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdNodeByIp() *Command {
	handler := func() {
		m.nodeMgr.GetNodeByIp()
	}
	cmd := &Command{
		Desc:    "根据ip查询node",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdArea() *Command {
	handler := func() {
		util.Province2Area(m.config)
	}
	cmd := &Command{
		Desc:    "省份转区域",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdCk() *Command {
	handler := func() {
		m.resources.Ck.RunCk()
	}
	cmd := &Command{
		Desc:    "查询clickhouse",
		NeedCK:  true,
		Handler: handler,
	}
	return cmd
}

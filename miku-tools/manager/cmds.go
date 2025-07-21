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
		Desc:    "qn http客户端, -uid <uid> -method <method(默认为GET)> -addr <url> -body <body>",
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
		Desc: "请求playcheck 302接口, -https <true/false> -domain <domain> " +
			"-bucket <bucket, 默认live> -stream <stream> -format <format, 默认flv> -sched_ip <sched_ip, 默认xs3427> " +
			"-user <user, 默认iqiyi> -node <node, 默认vdn-jsyz1-dls-1-9> -conn_id <conn_id, 默认12345678abcdef> -ip <clientIp>",
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

func (m *CommandManager) CmdRe() *Command {
	handler := func() {
		util.Re(m.config.Pattern, m.config.Replace, m.config.Raw)
	}
	cmd := &Command{
		Desc:    "测试的go的正则表达式, -pattern <pattern> -replace <replace> -raw <raw>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdHy() *Command {
	handler := func() {
		users.HyAuth(m.config)
	}
	cmd := &Command{
		Desc:    "虎牙时间戳防盗链, -stream <stream>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdBw() *Command {
	handler := func() {
		m.nodeMgr.BwStatistics()
	}
	cmd := &Command{
		Desc:               "统计剩余带宽情况",
		Handler:            handler,
		NeedIpParser:       true,
		NeedRedis:          true,
		NeedNodeInfo:       true,
		NeedMongo:          true,
		NeedNodeFilterInfo: true,
	}
	return cmd
}

func (m *CommandManager) CmdDumpNodes() *Command {
	handler := func() {
		m.nodeMgr.DumpNodes()
	}
	cmd := &Command{
		Desc:         "dump nodes并上传",
		Handler:      handler,
		NeedRedis:    true,
		NeedNodeInfo: true,
	}
	return cmd
}

func (m *CommandManager) CmdImportNodes() *Command {
	handler := func() {
		m.nodeMgr.WriteNodesToRedis()
	}
	cmd := &Command{
		Desc:         "从/tmp/allnodes.json导入节点信息, 写入redis",
		Handler:      handler,
		NeedRedis:    true,
		NeedNodeInfo: true,
	}
	return cmd
}

func (m *CommandManager) CmdCover() *Command {
	handler := func() {
		m.nodeMgr.NodesCover()
	}
	cmd := &Command{
		Desc:         "覆盖节点信息",
		Handler:      handler,
		NeedRedis:    true,
		NeedNodeInfo: true,
		NeedIpParser: true,
	}
	return cmd
}

func (m *CommandManager) CmdLoopPlaycheck() *Command {
	handler := func() {
		m.miku.LoopPlaycheck()
	}
	cmd := &Command{
		Desc:    "对playcheck接口进行拨测, 遍历每个省份*isp的组合, 测试playcheck返回的情况",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdLocate() *Command {
	handler := func() {
		m.Locate()
	}
	cmd := &Command{
		Desc:         "查询ip的位置信息, -ip <ip>",
		Handler:      handler,
		NeedIpParser: true,
	}
	return cmd
}

package manager

import (
	"mikutool/miku"
	mikumock "mikutool/miku/mock"
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
		Desc:    "qn http客户端, -uid <uid> -method <method(默认为GET)> -addr <url> -body <body> -header <key: value>",
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
		Desc: "请求playcheck 302接口, -https <true/false> -domain <domain> -app <app, 默认live> protocol <protocol, 默认flv> " +
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

func (m *CommandManager) CmdIpv6Nodes() *Command {
	handler := func() {
		m.nodeMgr.GetIpv6Nodes()
	}
	cmd := &Command{
		Desc:         "获取ipv6节点列表",
		Handler:      handler,
		NeedRedis:    true,
		NeedNodeInfo: true,
		NeedMongo:    true,
		NeedIpParser: true,
	}
	return cmd
}

func (m *CommandManager) CmdReport() *Command {
	handler := func() {
		m.streamMgr.NodeStreamReport()
	}
	cmd := &Command{
		Desc: "上报节点流信息, -node <node, default: 1-9> -online_num <online_num, default: 10> " +
			"-domain <domain, default> -app <app, default> -bucket <bucket, default> -stream <stream, default> " +
			"-ip <ip, default>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdIpv6Records() *Command {
	handler := func() {
		m.nodeMgr.GetIpv6DnsRecords()
	}
	cmd := &Command{
		Desc:         "获取ipv6 dns记录列表, -name <name> -domain <domain>",
		Handler:      handler,
		NeedNodeInfo: true,
		NeedDnsPod:   true,
	}
	return cmd
}

func (m *CommandManager) CmdTy() *Command {
	handler := func() {
		m.miku.TingYunErrNodes()
	}
	cmd := &Command{
		Desc:    "获取听云原始数据,统计ton n异常节点, -key <key> -task <task, default>, -start <start, default>, -end <end, default>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdMockTingyun() *Command {
	handler := func() {
		mikumock.MockTingyun()
	}
	cmd := &Command{
		Desc:    "模拟听云API服务",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdPathquery() *Command {
	handler := func() {
		m.miku.Pathquery()
	}
	cmd := &Command{
		Desc: "请求pathquery API, -conn_id <conn_id, default> -stream <stream, default> -domain <domain, default> " +
			"-area <area, default> -isp <isp, default> -bucket <bucket, default> -user <user, default> -sched_ip " +
			"<sched_ip, default> -format <format, default> -origin <origin, default>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdGetpcdn() *Command {
	handler := func() {
		m.miku.GetPCDN()
	}
	cmd := &Command{
		Desc:    "并发获取PCDN节点,测试lived性能, -n <n, default: 100>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdDy500() *Command {
	handler := func() {
		users.Dy500(m.config)
	}
	cmd := &Command{
		Desc:    "获取斗鱼500状态码百分比，画图表",
		Handler: handler,
	}
	return cmd
}

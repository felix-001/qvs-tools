package manager

import (
	"mikutool/miku"
	mikumock "mikutool/miku/mock"
	"mikutool/miku/staging"
	"mikutool/miku/users"
	"mikutool/public/util"
	"mikutool/qvs"
	"mikutool/qvs/mock"
	qvsStag "mikutool/qvs/staging"
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
		Desc: "请求playcheck 302接口, -https <true/false> \n-domain <domain> \n-app <app, 默认live> \n-protocol <protocol, 默认flv> \n" +
			"-bucket <bucket, 默认live> \n-stream <stream> \n-format <format, 默认flv> \n-sched_ip <sched_ip, 默认xs3427> \n" +
			"-user <user, 默认iqiyi> \n-node <node, 默认vdn-jsyz1-dls-1-9> \n-conn_id <conn_id, 默认12345678abcdef> \n-ip <clientIp> \n" +
			"-qn_test_url <streamd向lived请求playcheck的domain>",
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
		Desc:       "获取ipv6 dns记录列表, -name <name> -domain <domain>",
		Handler:    handler,
		NeedDnsPod: true,
		//NeedNodeInfo: true,
		//NeedDnsPod:   true,
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
		Desc: "请求pathquery API\n" +
			"如果指定-local, 则需要指定-node, -ip可选指定, 如果-ip不指定, 则传给调度的x-real-ip的header是空值\n" +
			" -conn_id <conn_id, default>\n -stream <stream, default>\n -domain <domain, default>\n " +
			"-area <area, default>\n -isp <isp, default>\n -bucket <bucket, default>\n -user <user, default>\n " +
			"-sched_ip <sched_ip, default>\n -node <node>\n -format <format, default>\n -origin <origin, default>\n " +
			"-ip <指定clientip>\n -skip <需要skip的节点>\n -random <如果指定了-area和-isp, 选点失败, 则默认会随机选个点>\n " +
			"-app <指定传入的playurl的app,如果不传，默认使用-bucket指定的参数>",
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

func (m *CommandManager) CmdTestdns() *Command {
	handler := func() {
		staging.TestDns(m.config, &m.resources)
	}
	cmd := &Command{
		Desc:    "httpdns接口拨测",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdTalk() *Command {
	handler := func() {
		qvs.TalkApi(m.config)
	}
	cmd := &Command{
		Desc:    "qvs对讲测试, -nsid <nsid> -gbid <gbid> -silence <true/false, 是否发送静音数据>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdTestiqiyi() *Command {
	handler := func() {
		staging.TestIqiyi(m.config, &m.resources)
	}
	cmd := &Command{
		Desc:         "测试iqiyi返点",
		Handler:      handler,
		NeedIpParser: true,
	}
	return cmd
}

func (m *CommandManager) CmdTestSip() *Command {
	handler := func() {
		staging.TestSip(m.config)

	}
	cmd := &Command{
		Desc:    "测试sip拨测, qvs-sip可能由于gb设备重启导致的崩溃问题, 0x556710094386 in SrsResourceManager::do_clear() src/app/srs_app_conn.cpp:367",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdProxy() *Command {
	handler := func() {
		miku.Proxy(m.config)
	}
	cmd := &Command{
		Desc:    "http代理,接收http请求,转发给upstream,相当于是一个代理,具体逻辑以及html是由upstream返回",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdTestHy() *Command {
	handler := func() {
		staging.TestHy(m.config, &m.resources)
	}
	cmd := &Command{
		Desc:         "分析虎牙投递的质量数据",
		Handler:      handler,
		NeedIpParser: true,
	}
	return cmd
}

func (m *CommandManager) CmdTestHy1() *Command {
	handler := func() {
		staging.TestHy1(m.config)
	}
	cmd := &Command{
		Desc:    "测试虎牙",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdQos() *Command {
	handler := func() {
		miku.Qos(m.config, &m.resources)
	}
	cmd := &Command{
		Desc:         "miku qos, upstream, 查询ck质量数据，返回html页面， 包含各种图表",
		Handler:      handler,
		NeedIpParser: true,
	}
	return cmd
}

func (m *CommandManager) CmdHlsPlay() *Command {
	handler := func() {
		miku.HlsPlay(m.config)
	}
	cmd := &Command{
		Desc:    "HLS播放",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdTestDev() *Command {
	handler := func() {
		qvsStag.TestDev(m.config)
	}
	cmd := &Command{
		Desc:    "更新qvs设备alarmTypesForSnap",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdProcMonitor() *Command {
	handler := func() {
		qvs.ProcMonitor(m.config)
	}
	cmd := &Command{
		Desc:    "监控到进程重启发送邮件，-process=<进程名> -smtp_host <SMTP服务器地址> -smtp_port <SMTP端口，默认587> -smtp_user <SMTP用户名> -smtp_pass <SMTP密码> -mail_from <发件人邮箱> -smtp_use_tls <是否使用TLS，默认true> -to <收件人邮箱，多个用逗号分隔> -subject <邮件主题> -body <邮件内容> -type <邮件类型：report/alert，默认report>",
		Handler: handler,
	}
	return cmd
}

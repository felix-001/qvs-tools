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
		Desc: "七牛鉴权 HTTP 客户端，自动计算 Qiniu Authorization 签名并发起请求，响应以格式化 JSON 输出\n" +
			"\n" +
			"鉴权方式(三选一，最终需有 ak/sk):\n" +
			"  -ak <ak> -sk <sk>              直接指定 ak/sk\n" +
			"  -uid <uid>                     通过 qconf 按 uid 查询 ak/sk(需配置 acc 账号文件)\n" +
			"  -user <user>                   从 /usr/local/etc/<user>.txt 读取 ak,sk\n" +
			"                                 可选值: mikutest, mikuonline, qvsmiku, qvs, admin, gray, vzan, qa, jfcs\n" +
			"\n" +
			"请求参数:\n" +
			"  -method <method>               HTTP 方法，默认 GET；若传了 -body 且未指定 -method，则默认 POST\n" +
			"  -addr <url>                    请求地址(与 -path 二选一，传 -path 时会忽略 -addr)\n" +
			"  -body <body>                   请求体(JSON 字符串)，非空时会自动添加 content-type: application/json\n" +
			"  -header <key: value>           自定义请求头，可多次指定\n" +
			"  -detail                        输出详细日志(含请求/响应头、ak/sk 等)\n" +
			"\n" +
			"快捷路径(-path 模式，自动拼接 URL，需配合 -user 选择后端域名):\n" +
			"  -path domain                   查询域名配置，需 -bucket -domain\n" +
			"  -path bucket                   查询 bucket 配置，需 -bucket\n" +
			"  -path roominfo                 查询房间信息，需 -bucket -id(roomid)\n" +
			"  -path listroom                 列出房间，需 -bucket，可选 -offset -limit\n" +
			"  -path userinfo                 查询用户信息，需 -bucket -id(roomid) -uid\n" +
			"  -path listuser                 列出用户，需 -bucket -id(roomid)，可选 -offset -limit\n" +
			"  -path deleteuser               删除用户(DELETE)，需 -bucket -id(roomid) -uid\n" +
			"  -path deleteroom               删除房间(DELETE)，需 -bucket -id(roomid)\n" +
			"  -path authrti                  鉴权 RTI，需 -bucket\n" +
			"  -path wm                       水印模板，PATCH/GET/DELETE 时需 -id\n" +
			"  -path upwm                     上传水印图片，DELETE 时需 -name\n" +
			"  -path listwm                   列出水印模板\n" +
			"  -path codec                    编解码模板\n" +
			"\n" +
			"-user 对应的后端域名:\n" +
			"  gray/qa(默认)                  mls-test.cn-east-1.qiniumiku.com\n" +
			"  mikutest                       mls.cn-east-1.jfcs.qiniu.io\n" +
			"  mikuonline/qvsmiku             mls.cn-east-1.qiniumiku.com\n" +
			"  qvs                            qiniuapi.com\n" +
			"\n" +
			"示例:\n" +
			"  miku -cmd http -ak xxx -sk xxx -method POST -addr http://miku-lived.dooquuequezi.com/api/v1/centerauth \\\n" +
			"    -body '{\"bucket\": \"test-bucket\"}' -header 'content-type:application/json' -detail\n" +
			"  miku -cmd http -user mikutest -path domain -bucket mybucket -domain test.example.com\n" +
			"  miku -cmd http -user qa -body '{\"source_path_query_sched\": \"v2\"}' -bucket liyqtest -detail -method PATCH" +
			"  miku -cmd http -user qa -path listroom -bucket mybucket -offset 0 -limit 20",
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
			"-port <port, 默认6060>\n" +
			"-qn_test_url <streamd向lived请求playcheck的domain>\n-head_req",
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
		util.DownloadIpdb(m.config)
	}
	cmd := &Command{
		Desc: "使用 ipdb 配置(ipdb.ips_source_param)中的 remote_url 从 kodo 下载 ipdb 文件到 /tmp 目录\n" +
			"  依赖配置项 ak/sk/remote_url/retry_count, 下载后保存为 /tmp/<时间戳>.ipdb",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdSign() *Command {
	handler := func() {
		util.SignResource(m.config)
	}
	cmd := &Command{
		Desc:    "kodo 私有资源签名, -uid <uid> -key <key> -domain <domain>",
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
			"-ip <ip, default> -bw <bandwidth>",
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

func (m *CommandManager) CmdDns() *Command {
	handler := func() {
		m.miku.Dns()
	}
	cmd := &Command{
		Desc: "dump dns记录列表, -doman <domain> -host <host, 可选> -del <是否删除, default: false> -id <record id> \n\texample: " +
			"-domain \"mikudns.com\"  -host \"qn-kuai-flv-yq.njyqkj0ksyz.com.subscribe\"\n" +
			"\t如果不指定-host则是获取所有记录",
		Handler:    handler,
		NeedDnsPod: true,
	}
	return cmd
}

func (m *CommandManager) CmdDnslog() *Command {
	handler := func() {
		m.miku.DnsLog()
	}
	cmd := &Command{
		Desc:       "获取dns操作日志(全量), -domain <domain>",
		Handler:    handler,
		NeedDnsPod: true,
	}
	return cmd
}

func (m *CommandManager) CmdTy() *Command {
	handler := func() {
		m.miku.TingYunErrNodes()
	}
	cmd := &Command{
		Desc: "获取听云原始数据,分析再缓冲时间异常IP\n" +
			"  -key <authkey>        听云鉴权key(也可在配置文件tingyun.auth_key中配置)\n" +
			"  -task <taskId>        任务ID(也可在配置文件tingyun.task_id中配置)\n" +
			"  -start <startTime>    开始时间,格式: 2006-01-02 15:04 (默认24小时前)\n" +
			"  -end <endTime>        结束时间,格式: 2006-01-02 15:04 (默认当前时间)\n" +
			"  -filter_ip <ip>       过滤IP(监测点IP或目标主机IP,可选)\n" +
			"  异常判定: 再缓冲时间>30s为异常记录, 异常率>60%为异常IP",
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

func (m *CommandManager) CmdOrigin() *Command {
	handler := func() {
		m.miku.Origin()
	}
	return &Command{
		Desc: "回源排查：遍历节点和流，查询 Trino internal-player 日志\n" +
			"  -f <file>           节点列表文件(首列为 nodeid，跳过表头)\n" +
			"  -t <file>           流列表文件(首列为 streamname，跳过表头)\n" +
			"  -day <YYYYMMDD>     查询日期(默认: 20260701)\n" +
			"  -hour <HH>          查询小时(默认: 14)\n" +
			"  -app <appname>      appname(默认: maozhua)\n" +
			"  -output <file>      输出文件(默认: origin_result_<ts>.jsonl)\n" +
			"  -addr <url>         节点查询 API(默认: http://miku-lived.qiniuapi.com:2240/v1/runtime/nodes)\n" +
			"\n" +
			"命中记录写入 JSONL，每行包含 nodeId、streamId、sql、results\n" +
			"example:\n" +
			"  miku -cmd origin -f nodes.txt -t streams.txt -day 20260701 -hour 14",
		Handler: handler,
	}
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

func (m *CommandManager) CmdPublish() *Command {
	handler := func() {
		miku.Publish(m.config)
	}
	cmd := &Command{
		Desc:    "发布, -env <env, default: online, 可选值online/jfcs>",
		Handler: handler,
	}
	return cmd
}

func (m *CommandManager) CmdFilternode() *Command {
	handler := func() {
		m.nodeMgr.Filternode()
	}
	return &Command{
		Desc:         "过滤节点",
		Handler:      handler,
		NeedIpParser: true,
		NeedNodeInfo: true,
	}
}

func (m *CommandManager) CmdBlacklist() *Command {
	handler := func() {
		m.nodeMgr.DownloadNodeStatus(m.config)
	}
	return &Command{
		Desc:    "下载节点状态",
		Handler: handler,
	}
}

func (m *CommandManager) CmdPanic() *Command {
	handler := func() {
		m.miku.TracePanic(m.config)
	}
	return &Command{
		Desc:    "trace panic",
		Handler: handler,
	}
}

func (m *CommandManager) CmdIploc() *Command {
	handler := func() {
		m.miku.IpLoc()
	}
	return &Command{
		Desc:         "ip loc",
		Handler:      handler,
		NeedIpParser: true,
	}
}

func (m *CommandManager) CmdNginxLog() *Command {
	handler := func() {
		m.miku.NginxLogSearch(m.config)
	}
	return &Command{
		Desc:    "在多个节点上并行搜索 nginx 日志, -path <日志目录> -pattern <文件匹配模式> -query <搜索关键词>",
		Handler: handler,
	}
}

func (m *CommandManager) CmdRegister() *Command {
	handler := func() {
		m.miku.StreamRegister()
	}
	return &Command{
		Desc: "stream register\n" +
			"  -bucket <bucket>     空间 (默认: liyqtest)\n" +
			"  -stream <stream>     流ID/key (默认: teststream)\n" +
			"  -node <nodeId>       nodeID (必填，默认: vdn-jsyz1-dls-1-87)\n" +
			"  -url <url>           完整推流url (必填， 默认: rtmp://<domain>/liyqtest/teststream)\n" +
			"  -ip <ip>             请求IP (默认: 114.230.94.166)\n" +
			"  -conn_id <id>        connectId (默认: 置空则自动生成)\n" +
			"  -domain <domain>     推流域名 (默认: liyqtest.com)\n" +
			"  -raw_app <app>       原始app名，映射为 rawUrl (可选，默认空)\n" +
			"  -sched_ip <ip>       调度服务地址 (默认: 10.34.146.62，请求发往 http://<sched_ip>:6060)\n" +
			"\n" +
			"隐式字段: localAddr 自动设为 <ip>:8080\n" +
			"example:\n" +
			"  miku -cmd register -bucket test-bucket -stream teststream -node node1 -domain test.com -url http://..." +
			" -protocol live -ip 1.2.3.4 -conn_id abc123",
		Handler: handler,
	}
}

/*
func (m *CommandManager) CmdPacket() *Command {
	handler := func() {
		m.miku.Packet()
	}
	return &Command{
		Desc:    "packet",
		Handler: handler,
	}
}
*/

func (m *CommandManager) CmdSsh() *Command {
	handler := func() {
		m.miku.WsExec(m.config)
	}
	return &Command{
		Desc: "通过 GoTTY WebSocket 在远程节点执行命令\n" +
			"  -node <nodeId>       节点 ID (必需)\n" +
			"  -query <command>     要执行的命令，默认 hostname\n" +
			"  -t <timeout_sec>     超时秒数，默认 30 (yaml: ws_exec.timeout_sec)\n" +
			"\n" +
			"yaml 配置项 (ws_exec):\n" +
			"  ws_host              WebSocket 主机\n" +
			"  admin_host           管理后台主机 (获取 usertoken)\n" +
			"  usertoken_path       usertoken API 路径\n" +
			"  auth                 管理后台 Authorization (可选)\n" +
			"  auth_file            缓存的 auth 文件路径\n" +
			"  user_file/pass_file  LinkCloud LDAP 账号密码文件\n" +
			"  totp_bin/totp_profile TOTP 命令及 profile\n" +
			"  login_url/target_url SSO 登录页与目标页 (可选)\n" +
			"  timeout_sec/columns/rows\n" +
			"\n" +
			"鉴权优先级: LINKCLOUD_AUTH 环境变量 > ws_exec.auth > auth_file > SSO 自动登录\n" +
			"example:\n" +
			"  miku -cmd ssh -node vdn-xxx -query hostname\n" +
			"  miku -cmd ssh -node vdn-xxx -query 'uname -a' -t 60",
		Handler: handler,
	}
}

func (m *CommandManager) CmdTimeout() *Command {
	handler := func() {
		csvPath := "/Users/liyuanquan/Downloads/sqllab_liyqkodofsagent_20260715T060745.csv"
		if m.config.Path != "" {
			csvPath = m.config.Path
		}
		m.miku.TimeoutStat(csvPath)
	}
	return &Command{
		Desc:    "读取 timeout CSV 日志，按 host 聚合统计条目数（从大到小），可用 -path 指定 CSV 文件路径",
		Handler: handler,
	}
}

func (m *CommandManager) CmdNiulink() *Command {
	handler := func() {
		m.miku.Niulink(m.config)
	}
	return &Command{
		Desc:    "niulink",
		Handler: handler,
	}
}

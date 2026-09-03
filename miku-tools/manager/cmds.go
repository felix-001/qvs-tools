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
		Desc: "请求 playcheck 302 接口，向调度服务 POST /api/v1/playcheck，打印格式化 JSON 响应\n" +
			"\n" +
			"请求目标:\n" +
			"  -sched_ip <ip>                 调度服务地址 (默认: 10.34.146.62)\n" +
			"  -port <port>                   调度服务端口 (默认: 6060)\n" +
			"\n" +
			"播放参数:\n" +
			"  -https                         使用 https 拼播放 URL (默认: false，即 http)\n" +
			"  -domain <domain>               播放域名 (默认: push.liyqtest.com)\n" +
			"  -app <app>                     app 名 (默认: live)\n" +
			"  -bucket <bucket>               空间 (默认: liyqtest)\n" +
			"  -stream <stream>               流名/key (默认: teststream)\n" +
			"  -format <format>               播放格式/后缀 (默认: flv)\n" +
			"  -protocol <protocol>           协议字段 (默认: flv)\n" +
			"\n" +
			"节点与客户端:\n" +
			"  -node <node>                   节点 ID (默认: ...-vdn-jsyz1-dls-1-87)\n" +
			"  -ip <clientIp>                 单次模式客户端 IP，写入 remote 为 <ip>:8080 (默认: 114.230.94.166)\n" +
			"  -user <user>                   用户标识 (可选)\n" +
			"\n" +
			"调用模式:\n" +
			"  -n <n>                         循环次数 (默认: 0)\n" +
			"    n=0  单次模式: 使用 -ip 调用一次 playcheck\n" +
			"    n>0  批量模式: 从 /tmp/ips.txt 读取 IP 列表(每行一个)，\n" +
			"         遍历每个 IP，对每个 IP 调用 playcheck 共 n 次\n" +
			"\n" +
			"可选参数:\n" +
			"  -qn_test_url <domain>          追加到播放 URL 的 qnTestUrl，streamd 向 lived 请求 playcheck 时使用的 domain\n" +
			"  -head_req                      使用 HEAD 方法请求 (默认: GET)\n" +
			"  -player <player>               追加 player 查询参数\n" +
			"  -query <query>                 追加自定义查询串\n" +
			"  -redirect                      使用 127.0.0.1 形式的 redirect 播放 URL\n" +
			"  -internal                      使用 internal 形式的播放 URL\n" +
			"\n" +
			"说明: conn_id 每次请求自动随机生成；播放 URL 形如 <scheme>://<domain>/<app>/<stream>.<format>",
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

func (m *CommandManager) CmdIpdb() *Command {
	handler := func() {
		util.DownloadIpdb(m.config)
	}
	cmd := &Command{
		Desc: "下载 ipdb 文件，ak/sk 从配置 ipdb.ips_source_param 读取\n" +
			"  -domain <domain>   下载域名（可选，默认 ipipfile.qbox.net）\n" +
			"  -name <ipv4|ipv6>  指定下载 ipv4 或 ipv6，不传则两者都下\n" +
			"  -path <path>       保存路径（可选）；默认用配置 local_url，再否则 /tmp/neo.<name>.ipdb\n" +
			"\n" +
			"下载地址:\n" +
			"  ipv4: http://<domain>/neo.ipv4.ipdb\n" +
			"  ipv6: http://<domain>/neo.ipv6.ipdb\n" +
			"\n" +
			"example:\n" +
			"  miku -cmd ipdb -name ipv4\n" +
			"  miku -cmd ipdb -name ipv6 -path /tmp/neo.ipv6.ipdb\n" +
			"  miku -cmd ipdb -domain ipipfile.qbox.net\n" +
			"  miku -cmd ipdb",
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

func (m *CommandManager) CmdNodelog() *Command {
	handler := func() {
		miku.NodeLog(m.config)
	}
	cmd := &Command{
		Desc: "登录多个节点远程搜索日志：自动准备 miku，展开路径参数后执行 sipsearch\n" +
			"\n" +
			"参数:\n" +
			"  -node <节点[,节点...]>          节点 ID，多个节点用逗号分隔；每个节点并发执行\n" +
			"  -path <路径模板>                 日志路径模板，支持 ${service}，例如 /home/qboxserver/${service}/_package/run/auditlog/sip_dump\n" +
			"  -path_params <key=值[,值...]>    占位参数，例如 service=qvs-sip,qvs-sip1,qvs-sip2，会展开成多个日志路径\n" +
			"  -pattern <正则[,正则...]>       传给 sipsearch 的行匹配正则，多个正则用逗号分隔\n" +
			"  -exclude_pattern <正则[,正则...]> 排除正则；信令内任意一行命中任意一个则不输出\n" +
			"  -raw <文件名正则>                文件名过滤正则，默认 .*dump.*log.*\n" +
			"  -archive <文件名>                miku 压缩包文件名，默认 miku-1788147017.tar.gz\n" +
			"  -force                           无论远端是否已有 miku，都重新下载并覆盖\n" +
			"  -output <文件>                   将结果追加写入本地文件；不传则打印到终端\n" +
			"\n" +
			"远端流程: 不存在 /home/qboxserver/liyq/miku 时，创建目录、从 qupfile.cloudvdn.com 下载压缩包、解压并 chmod +x；\n" +
			"随后执行 /home/qboxserver/liyq/miku -cmd sipsearch -path <展开后的路径> -pattern <正则> -raw <文件名正则>。\n" +
			"\n" +
			"示例:\n" +
			"  miku -cmd nodelog -node vdn-a,vdn-b -path '/home/qboxserver/${service}/_package/run/auditlog/sip_dump' \\\n" +
			"    -path_params 'service=qvs-sip,qvs-sip1,qvs-sip2' -pattern 'Register,Catalog' -raw '.*dump.*log.*'\n" +
			"  miku -cmd nodelog -node vdn-a -path '/home/qboxserver/${service}/_package/run/auditlog/sip_dump' \\\n" +
			"    -path_params 'service=qvs-sip' -pattern 'Register,Catalog' -force -output /tmp/nodelog.result",
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
		Desc: "向 lived 发送 pathquery 请求，模拟播放调度流程，返回拉流源站路径\n" +
			"工作模式:\n" +
			"  默认模式: 先通过 sched API 获取 PCDN 节点，再向该节点发起 pathquery\n" +
			"  -local:   直连模式，跳过 sched 选点，直接向指定 -node 发起 pathquery（必须指定 -node）\n" +
			"            此模式下 -ip 可选，不指定则 X-Real-IP header 为空\n" +
			"参数说明:\n" +
			"  -stream <流名>           播放流名称（默认使用配置值）\n" +
			"  -domain <域名>           播放域名（默认使用配置值）\n" +
			"  -bucket <空间名>         Bucket 名称（默认使用配置值）\n" +
			"  -app <应用名>            playUrl 中的 app 字段，不传则使用 -bucket 的值\n" +
			"  -format <格式>           播放格式，如 flv/m3u8/slice（默认使用配置值）\n" +
			"  -user <用户>             用户标识（默认使用配置值）\n" +
			"  -area <区域>             选点区域，用于 sched API 筛选节点\n" +
			"  -isp <运营商>            选点运营商，用于 sched API 筛选节点\n" +
			"  -sched_ip <调度地址>     lived 调度服务地址（默认使用配置值）\n" +
			"  -node <节点ID>           指定目标节点（-local 模式下必填）\n" +
			"  -ip <客户端IP>           指定 X-Real-IP，模拟客户端来源 IP\n" +
			"  -origin <源站地址>       回源地址（format=slice 时会转换为 ex1 参数）\n" +
			"  -skip <节点ID列表>       逗号分隔，pathquery 时跳过这些节点\n" +
			"  -random                  当 -area/-isp 选点失败时，随机选取一个节点\n" +
			"  -conn_id <连接ID>        连接标识（默认自动生成随机字符串）",
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
		Desc: "在多个节点上并行搜索 nginx/qvs-sip 日志：先 ls 匹配文件，再并行 grep\n" +
			"\n" +
			"必需参数:\n" +
			"  -query <keyword>               搜索关键词(grep -E)\n" +
			"\n" +
			"节点列表(优先从 /tmp/nodes.txt 读取):\n" +
			"  每行格式: <nodeid>_<序号>，序号可为空\n" +
			"  读取成功时自动推导 path/pattern，无需再传 -path/-pattern:\n" +
			"    path    = /home/qboxserver/qvs-sip<序号>/_package/run/\n" +
			"    pattern = qvs-sip<序号>.log*\n" +
			"  例: vdn-xxx        -> qvs-sip/_package/run/ , qvs-sip.log*\n" +
			"      vdn-xxx_2      -> qvs-sip2/_package/run/ , qvs-sip2.log*\n" +
			"\n" +
			"回退模式(/tmp/nodes.txt 不存在时，使用内置节点列表):\n" +
			"  -path <dir>                    日志目录(必需)\n" +
			"  -pattern <glob>                文件匹配模式(必需，如 *.log*)",
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

func (m *CommandManager) CmdSipsearch() *Command {
	handler := func() {
		miku.SipSearch(m.config)
	}
	cmd := &Command{
		Desc: "从本地日志中搜索同时满足多个正则的完整信令，最多并发处理 10 个文件\n" +
			"\n" +
			"参数:\n" +
			"  -path <路径[,路径...]>          日志文件或目录，多个路径用逗号分隔；目录会递归遍历其下的文件\n" +
			"  -pattern <正则[,正则...]>      行匹配正则，多个正则用逗号分隔；同一信令必须全部匹配才会输出\n" +
			"  -exclude_pattern <正则[,正则...]> 排除正则；信令内任意一行命中任意一个则不输出\n" +
			"  -raw <文件名正则>               目录下文件名过滤正则，例如 .*dump.*log.*；不传则处理目录下所有普通文件\n" +
			"\n" +
			"处理规则:\n" +
			"  每行依次匹配所有 -pattern；遇到 <--------------------------------------------------------------------------------------------------->\n" +
			"  视为一个信令结束。只有该信令匹配了全部正则才打印完整内容，随后重置匹配状态。\n" +
			"  每个文件读取到内存后逐行处理，文件完成后输出文件处理日志。\n" +
			"\n" +
			"示例:\n" +
			"  # -path 传具体日志文件；-raw 对具体文件同样按文件名过滤\n" +
			"  miku -cmd sipsearch -path /tmp/sip.log -pattern 'start,stream_id=[^ ]+'\n" +
			"\n" +
			"  # -path 传目录；递归搜索目录下文件，并按文件名正则过滤\n" +
			"  miku -cmd sipsearch -path /tmp -pattern 'Register,Catalog' -raw '.*dump.*log.*'\n" +
			"\n" +
			"  # -path 传多个文件或目录，使用逗号分隔\n" +
			"  miku -cmd sipsearch -path /var/log/a,/var/log/b -pattern 'foo,bar' -raw '.*dump.*log.*'",
		Handler: handler,
	}
	return cmd
}

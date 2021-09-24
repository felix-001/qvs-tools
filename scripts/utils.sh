#!/bin/bash

# 从日志文件名提取日期
# qvs-rtp.log-0723143753 -> 0723143753
# prfix: qvs-rtp.log-
# rawString: qvs-rtp.log-0723143753
GetDateFromFileName() {
	prefix=$1
	rawString=$2

	prefixLen=${#prefix}
	date=${rawString:prefixLen}
	echo $date
}

# 从文件列表获取最新文件
# sep: 文件列表分隔符
# prefix: 文件名字日期部分之前的字符串(qvs-rtp.log-)
# rawString: 文件列表(一般是空格)
FindLatestLogFile() {
	sep=$1
	rawString=$2
	prefix=$3

	lastDate=""
	lastLog=""
	while IFS="$sep" read -ra ADDR; do
  		for i in "${ADDR[@]}"; do
			date=`GetDateFromFileName $prefix $i`
			if [[ $date > $lastDate ]];then
				lastDate=$date
				lastLog=$i
			fi
  		done
	done <<< "$rawString"
	echo $lastLog
}

# 从目录中查找最新文件
# path: 路径
# prefix: qvs-rtp.log-
FindLatestLogFromFolder() {
	path=$1
	prefix=$2

	fileList=`cd $path;ls $prefix*`
	sep=" "
	latestLog=`FindLatestLogFile "$sep" "$fileList" "$prefix"`
	echo $latestLog
}

# 获取服务最新日志
# service: qvs-rtp
GetLatestLogOfService() {
	service=$1

	prefix="$service.log-"
	path=~/$service/_package/run/
	latestLog=`FindLatestLogFromFolder $path $prefix`
	echo $latestLog
}

# 实时查看服务日志，过滤gbid
TraceServiceLogById() {
	service=$1
	id=$2

	latestLog=`GetLatestLogOfService $service`
	echo "logfile: ~/$service/_package/run/$latestLog"
	tail -f ~/$service/_package/run/$latestLog | grep $id
}

traceLog() {
	service=$1
	gbid=`cat ~/liyq/etc/gbid.conf`
	if [ $# == 2 ];then
		gbid=$2
	fi
	TraceServiceLogById $service $gbid	

}

# 查看qvs-sip日志，grep gbid
# 参数为gbid, 如果不传，默认gbid在配置文件
siplog() {
	if [ $# != 0 ];then
		traceLog qvs-sip $1
	else
		traceLog qvs-sip
	fi
}

# 查看qvs-rtp日志，grep gbid
# 参数为gbid, 如果不传，默认gbid在配置文件
rtplog() {
	if [ $# != 0 ];then
		traceLog qvs-rtp $1
	else
		traceLog qvs-rtp
	fi
}

# 查看qvs-server日志，grep gbid
# 参数为gbid, 如果不传，默认gbid在配置文件
srvlog() {
	if [ $# != 0 ];then
		traceLog qvs-server $1
	else
		traceLog qvs-server
	fi
}

srvApiBasePath="http://localhost:7275/v1"

deviceReq() {
	uid=$1
	nsid=$2
	gbid=$3
	cmd=$4
	data=$5

	url="$srvApiBasePath/namespaces/$nsid/devices/$gbid/$cmd"
	curl --location --request POST $url \
		--header "authorization: QiniuStub uid=$uid" \
		--header "Content-Type: application/json" \
		-d "$data"
}

streamGetReq() {
	uid=$1
	nsid=$2
	streamId=$3
	cmd=$4
	query=$5

	url="$srvApiBasePath/namespaces/$nsid/streams/$streamId/$cmd?$query"
	curl --location --request GET $url \
		--header "authorization: QiniuStub uid=$uid"
}

streamPostReq() {
	uid=$1
	nsid=$2
	gbid=$3
	cmd=$4
	data=$5

	url="$srvApiBasePath/namespaces/$nsid/streams/$gbid/$cmd"
	curl --location --request POST $url \
		--header "authorization: QiniuStub uid=$uid" \
		--header "Content-Type: application/json" \
		-d "$data"
}

# 历史流控制

# 查询录制记录
# $1 - uid
# $2 - nsid
# $3 - streamId
# $4 - start
# $5 - end
record-playback() {
	streamGetReq $1 $2 $3 recordhistories "start=$4&end=$5&line=30&marker="
}

# 发起语音对讲
# $1 - uid
# $2 - nsid
# $3 - gbid
# $4 - 传输模式tcp/udp
# $5 - 对讲协议版本2014/2016
# $6 - isV2, 是否使用v2版接口true/false
talk() {
	if [ $# != 6 ];then
		echo "usage: talk <uid> <nsid> <gbid> <tcp/udp> <2014/2016> <isV2:true/false>"
		echo "       默认调度到vdn-gdgzh-dls-1-11"
		return 0
	fi

	pcmaB64=`cat ~/liyq/etc/pcma.b64`
	data="{
		\"rtpAccessIp\":\"14.29.108.156\",
		\"transProtocol\":\"$4\",
		\"tcpModel\":\"sendrecv\",
		\"version\":\"$5\",
		\"isV2\":\"$6\",
		\"base64Audio\":\"$pcmaB64\"
	}"
	resp=`deviceReq $1 $2 $3 $cmd "$data"`
	echo $resp
}

# 自己的portal账户请求语音对讲
# $1 - gbid
# $2 - 传输模式，tcp/udp
# $3 - 对讲协议版本，2014/2016
talk-internal() {
	if [ $# != 3 ];then
		echo "usage: talk-internal <gbid> <tcp/udp> <2014/2016>"
		echo "       默认调度到vdn-gdgzh-dls-1-11"
		return 0
	fi
	talk 1381539624 2xenzw72izhqy $1 $2 $3
}

# 发起拉流
# $1 - uid
# $2 - nsid
# $3 - gbid
# $4 - 传输模式tcp/udp
# $5 - chid, 不需要传是传""
invite() {
	if [ $# != 5 ];then
		echo "usage: invite <uid> <nsid> <gbid> <tcp/udp> <chid>"
		return 0
	fi
	uid=$1
	nsid=$2
	gbid=$3
	protocol=$4
	chid=$5

	data="{
		\"rtpAccessIp\":\"14.29.108.156\",
		\"rtpProto\":\"$protocol\",
		\"channels\":[\"$chid\"]
	}"
	deviceReq $uid $nsid $gbid "start" "$data"
}

# 发起拉历史流
# $1 - uid
# $2 - nsid
# $3 - gbid
# $4 - 传输模式tcp/udp
# $5 - chid, 不需要传是传""
# $6 - startTime
# $7 - endTime
invite-history() {
	if [ $# != 7 ];then
		echo "usage: invite <uid> <nsid> <gbid> <tcp/udp> <chid> <startTime> <endTime>"
		return 0
	fi
	uid=$1
	nsid=$2
	gbid=$3
	protocol=$4
	chid=$5
	start=$6
	end=$7

	data="{
		\"rtpAccessIp\":\"14.29.108.156\",
		\"rtpProto\":\"$protocol\",
		\"channels\":[\"$chid\"],
		\"start\":$start,
		\"end\":$end
	}"
	deviceReq $uid $nsid $gbid "start" "$data"
}

# 内部账号发起拉历史流
# $1 - gbid
# $2 - chid 传输模式tcp/udp
# $3 - chid
# $4 - start
# $5 - end
invite-history-internal() {
	if [ $# != 5 ];then
		echo "usage: invite <gbid> <tcp/udp> <chid> <start> <end>"
		echo "       默认调度到vdn-gdgzh-dls-1-11"
		return 0
	fi
	invite-history 1381539624 2xenzw72izhqy $1 $2 $3 $4 $5
}

# 内部账号发起拉流
# $1 - gbid
# $2 - chid 传输模式tcp/udp
# $3 - chid
invite-internal() {
	if [ $# != 3 ];then
		echo "usage: invite <gbid> <tcp/udp> <chid>"
		echo "       默认调度到vdn-gdgzh-dls-1-11"
		return 0
	fi
	invite 1381539624 2xenzw72izhqy $1 $2 $3
}

# 停止拉流
# $1 - uid
# $2 - nsid
# $3 - gbid
stopgb() {
	deviceReq $uid $nsid $gbid "stop" ""
}

rtpBaseUrl="http://localhost:2985/api/v1/gb28181?action="

# $1 - action
# $2 - id
# $3 - querys
rtpReq() {
	url="$sipBaseUrl$1&id=$2$3"
	curl $url
}

# dump ps流 
# $1 - 流id
dump() {
	rtpReq "dump_stream" $1 "&dump_ps=true"
}

sipApiBasePath="http://localhost:7279/api/v1/gb28181?action="
# $1 - action
# $2 - id
# $3 - querys
sipReq() {
	url="$sipApiBasePath$1&id=$2$3"
	curl $url
}

# 查询sip会话列表
# $1 - id
query-sess() {
	sipReq sip_query_session $1 ""
}

# 发布版本
# $1 - service name
publish() {
	if [ $# != 1 ];then
		echo "usage: publish <service>"
		echo "       qvs-server/qvs-sip/qvs-rtp/pili-flowd"
		return 0
	fi
	floy push $1
	floy version $1 | grep $1: | awk -F ',' '{print $1}' | xargs floy switch -f $1 _
	floy version $1 | grep $1: | awk -F ',' '{print $1}' | xargs floy run $1 restart.sh
}

# 发布版本指定node
# $1 - service name
# $2 - node
publish-node() {
	if [ $# != 2 ];then
		echo "usage: publish <service> <node>"
		echo "       qvs-server/qvs-sip/qvs-rtp/pili-flowd"
		return 0
	fi
	floy push $1 $2
	floy version $1 $2 | grep $1: | awk -F ',' '{print $1}' | xargs floy switch -f $1 _ $2
	floy run $1 restart.sh $2
}

# cpfrom 从节点拷贝到跳板机
# $1 - ndoeId
# $2 - file
cpf() {
	if [ $# != 2 ];then
        	echo "args <nodeId> <file>"
		return 0
	fi

	qscp qboxserver@$1:/home/qboxserver/liyq/$2 .
}

# cpto 从跳板机拷贝到节点
# $1 - ndoeId
# $2 - file
cpt() {
	if [ $# != 2 ];then
		echo "args <file> <node>"
		return 0
	fi

	qscp -r $1 qboxserver@$2:/home/qboxserver/liyq/
}

export gray='vdn-gdgzh-dls-1-11'

# 发布灰度环境
# $1 - service name
# $2 - node
pubish-gray() {
	if [ $# != 2 ];then
		echo "args <service> <node>"
		return 0
	fi
	floy push $1 $2
	floy version $1 $2 | grep $1: | awk -F ',' '{print $1}' | xargs floy switch -f $1 _ $2 
	floy run $1 restart.sh $2
}

publish-gray-srv() {
	publish-gray qvs-server jjh1449
}

publish-gray-sip() {
	publish-gray qvs-sip jjh1449
}

pubish-gray-rtp() {
	pubish-gray qvs-rtp $gray
}

# $1 - node
build-srs-cpto() {
	if [ $# != 1 ];then
		echo "args <node>"
		return 0
	fi
	cd ~/linking/srs/trunk
	git pull
	make -j8
	qscp -r objs/srs qboxserver@$1:/home/qboxserver/liyq/
}

build-srs-cpto-cs6() {
	build-srs-cpto cs6
	cs 6
}

build-srs-cpto-gray() {
	build-srs-cpto $gray
	qssh $gray
}

build-srs-cpto-jjh1449() {
	build-srs-cpto jjh1449
	qssh jjh1449
}

cp-flowd-to-gray() {
	cpt pili-flowd $gray
}

cp-srv-to-gray() {
	cpt qvs-server jjh1449
}

cp-srv-to-cs6() {
	cpt qvs-server cs6
}

cp-srs-to-jjh1449() {
	cpt srs jjh1449
}

cp-srs-to-gray() {
	cpt srs $gray
}
 
cp-srs-to-cs6() {
	cpt srs cs6
}

# 从一个节点拷贝到另一个节点
# $1 - file
# $2 - src node id
# $3 - dst node id
p2p() {
	if [ $# != 3 ];then
		echo "args <file> <src node id> <dst node id>"
		return 0
	fi

	file=$1
	srcNode=$2
	dstNode=$3
	qscp qboxserver@$srcNode:/home/qboxserver/liyq/$file .
	qscp $file qboxserver@$dstNode:/home/qboxserver/liyq/
}

export talknode='vdn-tz-tel-1-1'

srvconf() {
	vi ~/qvs-server/_package/qvs-server.conf
}

help() {
	path=`which utils.sh`
	grep .*\(\) $path
}
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
talk() {
	if [ $# != 5 ];then
		echo "usage: talk <uid> <nsid> <gbid> <tcp/udp> <2014/2016>"
		echo "\t默认调度到vdn-gdgzh-dls-1-11"
		return 0
	fi
	uid=$1
	nsid=$2
	gbid=$3
	protocol=$4
	version=$5

	pcmaB64=`cat ~/liyq/etc/pcma.b64`
	data="{
		\"rtpAccessIp\":\"14.29.108.156\",
		\"transProtocol\":\"$protocol\",
		\"tcpModel\":\"sendrecv\",
		\"version\":\"$version\",
		\"base64Audio\":\"$pcmaB64\"
	}"
	deviceReq $uid $nsid $gbid "talk" "$data"
}

# 发起拉流
# $1 - uid
# $2 - nsid
# $3 - gbid
# $4 - 传输模式tcp/udp
invite() {
	if [ $# != 4 ];then
		echo "usage: invite <uid> <nsid> <gbid> <tcp/udp>"
		return 0
	fi
	uid=$1
	nsid=$2
	gbid=$3
	protocol=$4

	data="{
		\"rtpAccessIp\":\"14.29.108.156\",
		\"rtpProto\":\"$protocol\"
	}"
	deviceReq $uid $nsid $gbid "start" "$data"
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


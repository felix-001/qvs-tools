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

# 发起语音对讲
# $1 - uid
# $2 - nsid
# $3 - gbid
# $4 - 传输模式tcp/udp
# $5 - 对讲协议版本2014/2016
talk() {
	if [ $# != 5 ];then
		echo "usage: talk <uid> <nsid> <gbid> <tcp/udp> <2014/2016>"
	uid=$1
	nsid=$2
	gbid=$3
	protocol=$4
	version=$5

	pcmaB64=`cat ~/liyq/etc/pcma.b64`
	url="$srvApiBasePath/namespaces/$nsid/devices/$gbid/talk"
	curl --location --request POST $url \
		--header "authorization: QiniuStub uid=$uid" \
		--header "Content-Type: application/json" \
		-d "{
			\"rtpAccessIp\":\"14.29.108.156\",
    			\"transProtocol\":\"$protocol\",
    			\"tcpModel\":\"sendrecv\",
    			\"version\":\"$version\",
			\"base64Audio\":\"$pcmaB64\"
		}"
}
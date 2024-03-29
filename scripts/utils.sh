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

srvApiBasePath="http://10.20.21.40:7275/v1"

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

# 获取某一个子串在字符串中的位置
# $1 - 原始字符串
# $2 - 子串
strindex() {
	x="${1%%$2*}"
	[[ "$x" = "$1" ]] && echo -1 || echo "${#x}"
}

# 指定首位限定字符串，获取子串
# $1 - 原始字符串
# $2 - 开始限定字符串
# $3 - 结尾限定字符串
# ex:
#     $1 = 111hello333 $2 = 111 $3 = 333
#     输出: hello
substr() {
	start=`strindex "$1" "$2"`
	start=$((start+${#2}))
	end=`strindex "$1" "$3"`
	length=$((end-start))
	res=${1:$start:$length}
	echo $res
}

# 发起语音对讲
# $1 - uid
# $2 - nsid
# $3 - gbid
# $4 - 传输模式tcp/udp
# $5 - 对讲协议版本2014/2016
# $6 - isV2, 是否使用v2版接口true/false
# $7 - protocol http/https
# $8 - schedIp，调度节点ip
talk() {
	if [ $# != 8 ];then
		echo "usage: talk <uid> <nsid> <gbid> <tcp/udp> <2014/2016> <isV2:true/false> <protocol:http/https> <schedIp>"
		echo "       默认调度到vdn-gdgzh-dls-1-11"
		return 0
	fi

	pcmaB64=`cat ~/liyq/etc/pcma.b64`
	data="{
		\"rtpAccessIp\":\"$8\",
		\"transProtocol\":\"$4\",
		\"tcpModel\":\"sendrecv\",
		\"version\":\"$5\",
		\"isV2\":$6,
		\"base64Audio\":\"$pcmaB64\"
	}"
	resp=`deviceReq $1 $2 $3 "talk" "$data"`
	echo $resp
	if [[ "x$resp" != "x" ]];then
		http=`echo $resp | jq -r '.audioSendAddrForHttp'`
		https=`echo $resp | jq -r '.audioSendAddrForHttps'`
		url=$http
		if [[ "$7" == "https" ]];then
			url=$https
		fi
		curl -v --location --request POST $url \
			--header "authorization: QiniuStub uid=$1" \
			--header "Content-Type: application/json" \
			-d "{\"base64_pcm\": \"$pcmaB64\"}"
	fi
}

# 自己的portal账户请求语音对讲
# $1 - gbid
# $2 - 传输模式，tcp/udp
# $3 - 对讲协议版本，2014/2016
# $4 - isV2, 是否使用v2版接口true/false
# $5 - protocol, http/https
# $6 - schedIp，调度节点ip
talk-internal() {
	if [ $# != 6 ];then
		echo "usage: talk-internal <gbid> <tcp/udp> <2014/2016> <isV2:true/false> <protocol:http/https> <schedIp>"
		echo "       默认调度到vdn-gdgzh-dls-1-11"
		return 0
	fi
	talk 1381539624 2xenzw72izhqy $1 $2 $3 $4 $5 $6
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

# 请求下载历史录像文件
# $1 - uid
# $2 - nsid
# $3 - gbid
# $4 - chid
# $5 - startTime
# $6 - endTime
# $7 - 传输模式tcp/udp
download() {
	if [ $# != 7 ];then
		echo "usage: download <uid> <nsid> <gbid> <chid> <startTime> <endTime> <tcp/udp>"
		return 0
	fi	
    	data="{
		\"channelId\":\"$4\",
    		\"start\":$5,
    		\"end\":$6,
    		\"rtpProto\":\"$7\",
		\"rtpAccessIp\":\"14.29.108.156\"
	}"
	deviceReq $1 $2 $3 "download" "$data"
}

# 自己的portal账户请求下载历史录像文件
# $1 - gbid
# $2 - chid
# $3 - startTime
# $4 - endTime
# $5 - 传输模式tcp/udp
download-inernal() {
	if [ $# != 5 ];then
		echo "usage: download-internal <gbid> <chid> <startTime> <endTime> <tcp/udp>"
		return 0
	fi
	download 1381539624 2xenzw72izhqy $1 $2 $3 $4 $5
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

rtpBaseUrl="http://127.0.0.1:2985/api/v1/gb28181?action="

# $1 - action
# $2 - id
# $3 - querys
rtpReq() {
	url="$rtpBaseUrl$1&id=$2$3"
	curl $url
}

# dump ps流 
# $1 - 流id
# $2 - 开：true 关：false
dump-ps() {
	if [ $# != 2 ];then
		echo "usage: dump-ps <streamId> <true/false>"
		return 0
	fi	
	rtpReq "dump_stream" $1 "&dump_ps=$2"
}

# dump 音频流 
# $1 - 流id
# $2 - 开：true 关：false
dump-audio() {
	if [ $# != 2 ];then
		echo "usage: dump-audio <streamId> <true/false>"
		return 0
	fi	
	rtpReq "dump_stream" $1 "&dump_audio=$2"
}

# dump 视频流 
# $1 - 流id
# $2 - 开：true 关：false
dump-video() {
	if [ $# != 2 ];then
		echo "usage: dump-video <streamId> <true/false>"
		return 0
	fi	
	rtpReq "dump_stream" $1 "&dump_video=$2"
}

sipApiBasePath="http://10.20.21.40:7279/api/v1/gb28181?action="
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
publish-gray() {
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

publish-gray-rtp() {
	publish-gray qvs-rtp $gray
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

export talknode='jjh236'

srvconf() {
	vi ~/qvs-server/_package/qvs-server.conf
}

help() {
	path=`which utils.sh`
	grep .*\(\) $path
}

qvssnap() {
	if [ $# != 3 ];then
		echo "<uid> <nsid> <streamid>"
		return 0
	fi
	uid=$1
	nsid=$2
	sid=$3
	cmd="snap"
	streamPostReq  $uid $nsid $sid $cmd
}

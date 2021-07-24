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

	fileList=`cd $path;ls $preifx*`
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

GetLatestQvsRtpLog() {
	latestLog=`GetLatestLogOfService qvs-rtp`
	echo $latestLog
}

GetLatestQvsSipLog() {
	latestLog=`GetLatestLogOfService qvs-sip`
	echo $latestLog
}

GetLatestQvsServerLog() {
	latestLog=`GetLatestLogOfService qvs-server`
	echo $latestLog
}

# 实时查看服务日志，过滤gbid
TraceServiceLogById() {
	service=$1
	id=$2

	latestLog=`GetLatestLogOfService $service`
	echo "logfile:$latestLog"
	tail -f ~/qvs-rtp/_package/run/$latestLog | grep $id
}

# 监控qvs-rtp日志，过滤gbid
MonitorRtpLog() {
	id=$1
	TraceServiceLogById qvs-rtp $id
}



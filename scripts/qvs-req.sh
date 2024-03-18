# 检查是否提供了正确的参数数量
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <path> [<body/uid> <uid>]"
    exit 1
fi

# 读取uid.conf文件中的UID
uid=$(cat /usr/local/etc/uid.conf)
uid=${uid%:} # 移除uid字符串的最后一个字符（如果有冒号）

# 根据参数数量和内容设置uid和body
if [ "$#" -eq 4 ]; then
    uid="$3"
    body="$4"
elif [ "$#" -eq 3 ]; then
    if [[ "$2" =~ ^[0-9]+$ ]]; then  # 检查第二个参数是否为数字
        uid="$2"
    else
        body="$2"
    fi
fi

# 从第一个参数中提取path
path="$1"

# 输出结果用于调试
echo "Path: $path"
echo "UID: $uid"
echo "Body: $body"

cmd="curl -s http://localhost:7275/v1$path"
cmd+=" --header 'authorization: QiniuStub uid=$uid'"
if [ -n "$body" ]; then
    cmd+=" $body"
fi


# ssh -t liyuanquan@10.20.34.27 "qssh bili-jjh9 \" curl -s http://localhost:7275/v1/namespaces --header 'authorization: QiniuStub uid=1381539624'  \""
ssh -t liyuanquan@10.20.34.27 "qssh bili-jjh9 \" $cmd  \"" | jq

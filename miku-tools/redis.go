package main

import (
	"context"
	"log"

	"bufio"
	"os"

	"github.com/redis/go-redis/v9"
)

func (s *Parser) Redis() {
	// 原有的 Redis 客户端初始化代码
	redisCli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      []string{"127.0.0.1:6380"},
		MaxRetries: 3,
		PoolSize:   30,
	})

	err := redisCli.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// 打开文件
	file, err := os.Open("/Users/liyuanquan/workspace/tmp/redis_export.txt")
	if err != nil {
		log.Fatalf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 创建一个新的 Scanner 来按行读取文件
	scanner := bufio.NewScanner(file)
	i := 0
	key := ""
	for scanner.Scan() {
		line := scanner.Text()
		//fmt.Println(line)
		if i%2 == 1 && key != "" {
			//log.Println("key:", key, "value", line)
			_, err = redisCli.HSet(context.Background(), "mik_netprobe_runtime_nodes_map", key, line).Result()
			if err != nil {
				log.Println("HSet failed", err)
			}
		}
		i++
		key = line
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("读取文件时出错: %v", err)
	}
}

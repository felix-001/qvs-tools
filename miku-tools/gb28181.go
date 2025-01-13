package main

import (
	"fmt"
	"net"
	"os"
)

func (p *Parser) Gb() {

	// 服务器地址
	serverAddr := fmt.Sprintf("%s:55068", p.conf.Ip)

	// 连接到服务器
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接到服务器失败: %v\n", err)
		return
	}
	defer conn.Close()

	// 要发送的数据包
	data := make([]byte, 1000000000) //[]byte("hello world, 111111111111111111111111111111111111111111111111111111")

	// 发送数据包
	for i := 0; i < 10000000; i++ {
		_, err = conn.Write(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "发送数据失败: %v\n", err)
			return
		}

		fmt.Println("数据发送成功", i)
		//time.Sleep(1 * time.Second)
	}
	/*
		for {
			// 读取服务器返回的数据
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Fprintf(os.Stderr, "读取数据失败: %v\n", err)
				return
			}

			fmt.Printf("服务器返回: %s\n", string(buf[:n]))
		}
	*/
}

package main

import (
	"log"
	"net"
	"os"
)

const (
	HOST = "localhost"
	PORT = "8080"
	TYPE = "tcp"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	udpServer, err := net.ResolveUDPAddr("udp", "101.133.131.188:5065")

	if err != nil {
		println("ResolveUDPAddr failed:", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, udpServer)
	if err != nil {
		println("Listen failed:", err.Error())
		os.Exit(1)
	}

	defer conn.Close()

	log.Println("send", os.Args[1])
	_, err = conn.Write([]byte(os.Args[1]))
	if err != nil {
		println("Write data failed:", err.Error())
		os.Exit(1)
	}

	received := make([]byte, 1024)
	_, err = conn.Read(received)
	if err != nil {
		println("Read data failed:", err.Error())
		os.Exit(1)
	}
	log.Println("received:", string(received))
}

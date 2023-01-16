package main

import (
	"log"
	"net"
	"strconv"
	"time"
)

func response(conn net.PacketConn, addr net.Addr, buf []byte) {
	log.Printf("time received: %v. Your message: %v!\n", time.Now().Format(time.ANSIC), string(buf))
	t, err := strconv.Atoi(string(buf))
	if err != nil {
		log.Println("parse buf err")
		return
	}
	log.Println("sleep", t, "second")
	time.Sleep(time.Second * time.Duration(t))
	log.Println("send resp")
	if _, err := conn.WriteTo(buf, addr); err != nil {
		log.Println("send resp err", err)
	}
}

func udpServer() {
	conn, err := net.ListenPacket("udp", ":port")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}
		go response(conn, addr, buf)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	udpServer()
}

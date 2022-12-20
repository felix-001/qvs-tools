package main

import (
	"log"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

var s = `
<?xml version="1.0" encoding="utf-8"?><Probe><Uuid>5FC99970-D38E-4024-8FC0-85DEED82D653</Uuid><Types>inquiry</Types></Probe>
`

func sendUDPMulticast(msg string, interfaceName string) ([]string, error) {
	c, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	defer c.Close()

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	p := ipv4.NewPacketConn(c)
	group := net.IPv4(239, 255, 255, 250)
	if err := p.JoinGroup(iface, &net.UDPAddr{IP: group}); err != nil {
		return nil, err
	}

	dst := &net.UDPAddr{IP: group, Port: 37020}
	data := []byte(msg)
	for _, ifi := range []*net.Interface{iface} {
		if err := p.SetMulticastInterface(ifi); err != nil {
			return nil, err
		}
		p.SetMulticastTTL(2)
		if _, err := p.WriteTo(data, nil, dst); err != nil {
			return nil, err
		}
	}

	if err := p.SetReadDeadline(time.Now().Add(time.Second * 2)); err != nil {
		return nil, err
	}

	var result []string
	for {
		b := make([]byte, 1024)
		n, _, _, err := p.ReadFrom(b)
		if err != nil {
			return result, err
		}
		result = append(result, string(b[0:n]))
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ss, err := sendUDPMulticast(s, "en0")
	if err != nil {
		log.Println(err)
	}
	log.Println(ss)
}

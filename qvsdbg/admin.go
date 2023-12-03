package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type Device struct {
	NodeId      string `json:"nodeId"`
	RtpAccessIp string `json:"rtpAccessIp"`
	NsId        string `json:"nsId"`
}

func getDevice(gbid string) (*Device, error) {
	cmd := fmt.Sprintf("curl -s --location --request GET http://10.20.76.42:7277/v1/devices/%s --header 'authorization: QiniuStub uid=1'", gbid)
	result, err := jumpboxCmd(cmd)
	if err != nil {
		log.Println(result, err, cmd)
		return nil, err
	}
	//log.Println("result:", result)
	device := &Device{}
	if err := json.Unmarshal([]byte(result), device); err != nil {
		return nil, err
	}
	return device, err
}

func getChannel(nsid, gbid, chid string) (*Device, error) {
	cmd := fmt.Sprintf("curl -s --location --request GET http://10.20.76.42:7277/v1/namespaces/%s/devices/%s/channels/%s --header 'authorization: QiniuStub uid=1'", nsid, gbid, chid)
	result, err := jumpboxCmd(cmd)
	if err != nil {
		log.Println(result, err, cmd)
		return nil, err
	}
	//log.Println("result:", result, cmd)
	device := &Device{}
	if err := json.Unmarshal([]byte(result), device); err != nil {
		return nil, err
	}
	return device, err
}

func (s *Parser) getInviteRtpNode(gbid, chid string) (string, error) {
	if chid == "" {
		dev, err := getDevice(gbid)
		if err != nil {
			return "", err
		}
		return s.getNodeByIP(dev.RtpAccessIp)
	}
	dev, err := getDevice(gbid)
	if err != nil {
		return "", err
	}
	ch, err := getChannel(dev.NsId, gbid, chid)
	if err != nil {
		return "", err
	}
	return s.getNodeByIP(ch.RtpAccessIp)
}

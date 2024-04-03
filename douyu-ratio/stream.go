package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type IpInfo struct {
	Ip        string `json:"ip" bson:"ip"`               // 服务IP地址
	OnlineNum uint32 `json:"onlineNum" bson:"onlineNum"` // 在线观看人数
	Bandwidth uint64 `json:"bandwidth" bson:"bandwidth"` // 下行服务带宽， 单位Byte/s
	HlsBytes  uint64 `json:"hlsBytes" bson:"hlsBytes"`
}

type PusherInfo struct {
	ConnectId string `json:"connectId" bson:"connectId"` // streamRegister时的connectId
	Type      string `json:"type" bson:"type"`           // publisher: 主动推流，puller: 被动回源
	Master    bool   `bson:"master" bson:"master"`       // true：主流or回源流， false: 备流
}

type PlayerInfo struct {
	Protocol string    `json:"protocol" bson:"protocol"`
	Ips      []*IpInfo `json:"ips" bson:"ips"`
}

type Profile struct {
	VideoCodec      string  `json:"videoCodec" bson:"videoCodec"`
	VideoRate       uint64  `json:"vidoeRate" bson:"vidoeRate"`
	VideoFps        float32 `json:"videoFps" bson:"videoFps"`
	VideoResolution string  `json:"videoResolution" bson:"videoResolution"`
	AudioCodec      string  `json:"audioCodec" bson:"audioCodec"`
	AudioRate       uint64  `json:"audioRate" bson:"audioRate"`
	AudioChannel    int     `json:"audioChannel" bson:"audioChannel"`
	AudioSample     int     `json:"audioSample" bson:"audioSample"`
}

type StreamInfoRT struct {
	Domain         string        `json:"domain" bson:"domain"`
	AppName        string        `json:"appName" bson:"appName"`
	StreamName     string        `json:"streamName" bson:"streamName"`
	Bucket         string        `json:"bucket" bson:"bucket"`
	Key            string        `json:"key" bson:"key"`
	Bandwidth      uint64        `json:"bandwidth" bson:"bandwidth"`
	RelayBandwidth uint64        `json:"relayBandwidth" bson:"relayBandwidth"` // 回源拉流带宽: 单位Byte/s
	Profile        *Profile      `json:"profile" bson:"profile"`
	Players        []*PlayerInfo `json:"player" bson:"player"`
	Pusher         []PusherInfo  `json:"pusher" bson:"pusher"`
	ConnectId      string        `json:"cid" bson:"cid"`
}

type NodeStreamInfo struct {
	Streams        []*StreamInfoRT `json:"streams"` // 该节点上所有在线流的信息
	Port           StreamdPorts    `json:"port"`    // streamd服务端口
	NodeId         string          `json:"nodeId"`
	LastUpdateTime int64           `json:"lastUpdateTime"` // 最后上报时间戳，单位：秒
}

func GetNodeStreams(nodeId string) *NodeStreamInfo {
	key := "stream_report_" + nodeId
	cmd := fmt.Sprintf("echo \"get %s\" | redis-cli-5 -x -h 10.20.54.24 -p 8200 -c --raw", key)
	data, err := JumpboxCmd(cmd)
	if err != nil {
		log.Fatalln(err)
	}

	if len(data) < 10 {
		return nil
	}
	//log.Println("data:", data)
	var nodeStreamInfo NodeStreamInfo
	if err = json.Unmarshal([]byte(data), &nodeStreamInfo); err != nil {
		log.Fatalln(err)
	}
	return &nodeStreamInfo
}

package main

import (
	"encoding/json"
	"log"
	"time"
)

type NodeStatus string

const (
	NODE_PENDING    NodeStatus = "pending"
	NODE_NORMAL     NodeStatus = "normal"
	NODE_DISABLED   NodeStatus = "disabled"
	NODE_UNKNOWN    NodeStatus = "unknown"
	NODE_ANY_STATUS NodeStatus = ""
)

type Ability struct {
	Can    bool `json:"can" bson:"can"`
	Frozen bool `json:"frozen" bson:"frozen"`
}

type IpIsp struct {
	Ip         string `bson:"ip" json:"ip"`
	Isp        string `bson:"isp" json:"isp"`
	Forbidden  bool   `bson:"forbidden,omitempty" json:"forbidden,omitempty"` // 单个 ip 封禁
	ExternalIP string `bson:"externalIP,omitempty" json:"externalIP,omitempty"`

	IsIPv6 bool `bson:"is_ipv6" json:"is_ipv6"`
}

type Node struct {
	Id           string             `bson:"_id" json:"id"`
	Idc          string             `bson:"idc" json:"idc"`
	Provider     string             `bson:"provider" json:"provider"`
	HostName     string             `bson:"host" json:"host"`
	BandwidthMbs float64            `bson:"bwMbps" json:"bwMbps"`
	LanIP        string             `bson:"lanIP" json:"lanIP"`
	Status       NodeStatus         `bson:"status" json:"status"`
	Abilities    map[string]Ability `bson:"abilities" json:"abilities,omitempty"` // fixme: key 形式待定
	IpIsps       []IpIsp            `bson:"ipisps" json:"ipisps"`
	Comment      string             `bson:"comment" json:"comment"`
	UpdateTime   int64              `bson:"updateTime" json:"updateTime"`
	IsDynamic    bool               `bson:"isDynamic" json:"isDynamic"`
	IsMixture    bool               `bson:"isMixture" json:"isMixture"`
	MachineId    string             `bson:"machineId" json:"machineId"`
	RecordTime   time.Time          `bson:"recordTime" json:"recordTime"`
	// fixme: oauth 相关信息记录
}

type IPStreamProbe struct {
	State      int     `json:"state"`      // 定义同 `StreamProbeState`
	Speed      float64 `json:"speed"`      // Mbps
	UpdateTime int64   `json:"updateTime"` // s

	// for sliding window
	SlidingSpeeds [10]float64 `json:"slidingSpeeds"`
	MinSpeed      float64     `json:"minSpeed"`
}

type RtIpStatus struct {
	IpIsp
	Interface  string  `json:"interface"`  // 网卡名称
	InMBps     float64 `json:"inMBps"`     // 入流量带宽，单位：MBps
	OutMBps    float64 `json:"outMBps"`    // 出流量带宽，单位：MBps
	MaxInMBps  float64 `json:"maxInMBps"`  // 下行建设带宽, 单位：MBps
	MaxOutMBps float64 `json:"maxOutMBps"` // 上行建设带宽，单位：MBps

	IPStreamProbe IPStreamProbe `json:"ipStreamProbe"`
}

type StreamdPorts struct {
	Http    int `json:"http" bson:"http"`       // http, ws
	Https   int `json:"https" bson:"https"`     // https, wss
	Wt      int `json:"wt" bson:"wt"`           // wt, quic
	Rtmp    int `json:"rtmp" bson:"rtmp"`       // rtmp
	Control int `json:"control" bson:"control"` // control
}

type RtNode struct {
	Node

	// runtime info
	RuntimeStatus string                   `json:"runtimeStatus"`
	Ips           []RtIpStatus             `json:"ips"`
	StreamdPorts  StreamdPorts             `json:"streamdPorts"`
	Services      map[string]ServiceStatus `json:"services"`
}

type ServiceStatus struct {
	CPU    float64 `json:"cpu"` // cpu usage percent
	RSS    int64   `json:"rss"` // resident set memory size
	FD     int     `json:"fd"`
	MaxFD  int     `json:"max_fd"`
	Uptime int64   `json:"uptime"`
}

func GetDynamicNodesData() []RtNode {
	s, err := JumpboxCmd("curl -s http://10.34.139.33:2240/v1/runtime/nodes?dynamic=true")
	if err != nil {
		log.Fatalln(err)
	}
	nodes := []RtNode{}
	if err := json.Unmarshal([]byte(s), &nodes); err != nil {
		log.Fatalln(err)
	}
	return nodes
}

func CheckNode(node RtNode) bool {
	if node.RuntimeStatus != "Serving" {
		return false
	}
	if node.StreamdPorts.Http <= 0 || node.StreamdPorts.Https <= 0 || node.StreamdPorts.Wt <= 0 {
		return false
	}
	ability, ok := node.Abilities["live"]
	if !ok || !ability.Can || ability.Frozen {
		return false
	}

	if _, ok = node.Services["live"]; !ok {
		return false
	}
	return true
}

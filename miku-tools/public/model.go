package public

import "github.com/qbox/mikud-live/common/model"

type CmdHandler func()

type CmdInfo struct {
	Handler CmdHandler
	Usage   string
	Depends []*bool
}

type DynamicRootNode struct {
	NodeId        string
	Forbidden     bool
	Err           string
	ForbiddenTime int64
}

type NodeDetail struct {
	OnlineNum uint32
	RelayBw   float64
	Bw        float64
	MaxBw     float64
	Ratio     float64
	RelayType uint32
	Protocol  string
	NodeId    string
}

type StreamInfo struct {
	//EdgeNodes    []string
	//EdgeNodes    map[string]*model.StreamInfoRT
	//RootNodes    []string
	//OfflineNodes []string
	//StaticNodes []string
	RelayBw       float64
	Bw            float64
	OnlineNum     uint32
	NodeStreamMap map[string]map[string]*model.StreamInfoRT // key1: node type(edge/root/offline/static) key2: node Id
}

type NodeUnavailableDetail struct {
	Start    string
	End      string
	Duration string
	Reason   string
	Detail   string
}

type StreamDetail struct {
	model.StreamInfoRT
	NodeId string
}

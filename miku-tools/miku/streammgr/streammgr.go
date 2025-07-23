package streammgr

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"

	schedModel "github.com/qbox/mikud-live/cmd/sched/model"
	commonModel "github.com/qbox/mikud-live/common/model"
)

type StreamMgr struct {
	conf *config.Config
}

func NewStreamMgr() *StreamMgr {
	return &StreamMgr{}
}

func GetNodesByStreamId(conf *config.Config) map[string]schedModel.StreamNodeDetailList {

	addr := fmt.Sprintf("http://10.34.146.62:6060/api/v1/bucket/%s/stream/%s/nodes",
		conf.Bucket, conf.Stream)
	resp, err := util.Get(addr)
	if err != nil {
		return nil
	}
	var nodesMap map[string]schedModel.StreamNodeDetailList
	if err := json.Unmarshal([]byte(resp), &nodesMap); err != nil {
		log.Println(err)
		return nil
	}
	return nodesMap
}

func (s *StreamMgr) SetConf(conf *config.Config) {
	s.conf = conf
}

func (s *StreamMgr) NodeStreamReport() {
	req := commonModel.StreamReportRequest{
		NodeId:    s.conf.Node,
		ConnectId: s.conf.ConnId,
		Streams: []*commonModel.StreamInfoRT{
			{
				Domain:     s.conf.Domain,
				AppName:    s.conf.App,
				StreamName: s.conf.Bucket + ":" + s.conf.Stream,
				Bucket:     s.conf.Bucket,
				Key:        s.conf.Stream,
				ConnectId:  s.conf.ConnId,
				Players: []*commonModel.PlayerInfo{
					{
						Protocol: s.conf.Protocol,
						Ips: []*commonModel.IpInfo{
							{
								Ip:        s.conf.Ip,
								OnlineNum: uint32(s.conf.OnlineNum),
							},
						},
					},
				},
			},
		},
	}

	bytes, err := json.Marshal(&req)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("req: %s\n", string(bytes))
	var resp commonModel.StreamReportResponse
	addr := fmt.Sprintf("http://%s:6060/api/v1/streamreport", s.conf.SchedIp)
	if err := util.Post(addr, string(bytes), &resp); err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("resp: %+v\n", resp)
}

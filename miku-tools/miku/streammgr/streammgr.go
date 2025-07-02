package streammgr

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"

	schedModel "github.com/qbox/mikud-live/cmd/sched/model"
)

type StreamMgr struct {
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

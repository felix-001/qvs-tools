package main

import (
	"encoding/json"
	"fmt"
	"log"
	"middle-source-analysis/util"

	"github.com/qbox/mikud-live/cmd/sched/model"
)

func (s *Parser) getNodesByStreamId() map[string]model.StreamNodeDetailList {

	addr := fmt.Sprintf("http://10.34.146.62:6060/api/v1/bucket/%s/stream/%s/nodes",
		s.conf.Bucket, s.conf.Stream)
	resp, err := util.Get(addr)
	if err != nil {
		s.logger.Error().Str("addr", addr).Err(err).Msg("get stream nodes")
		return nil
	}
	var nodesMap map[string]model.StreamNodeDetailList
	if err := json.Unmarshal([]byte(resp), &nodesMap); err != nil {
		log.Println(err)
		return nil
	}
	return nodesMap
}

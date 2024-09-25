package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/qbox/mikud-live/cmd/lived/common/util"
	"github.com/qbox/mikud-live/common/model"
)

func (s *Parser) getNodeAllStreams(nodeId string) (*model.NodeStreamInfo, error) {
	ctx := context.Background()
	val, err := s.RedisCli.Get(ctx, util.GetStreamReportRedisKey(nodeId)).Result()
	if err != nil {
		return nil, err
	}
	var nodeStreamInfo model.NodeStreamInfo
	if err = json.Unmarshal([]byte(val), &nodeStreamInfo); err != nil {
		log.Printf("[GetNodeStreams][Unmarshal], nodeId:%s, value:%s\n", nodeId, val)
		return nil, err
	}
	return &nodeStreamInfo, nil
}

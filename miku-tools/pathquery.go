package main

import (
	"encoding/json"
	"fmt"
	"log"
	"middle-source-analysis/util"

	"github.com/qbox/mikud-live/common/model"
)

func (s *Parser) PathqueryReq() {
	nodeId, _ := s.getPcdnFromSchedAPI(true, true)
	if nodeId == "" {
		s.logger.Info().Str("area", s.conf.Area).Str("isp", s.conf.Isp).Msg("get pcdn err")
		return
	}
	req := model.PathQueryRequest{
		Bucket:    s.conf.Bucket,
		Key:       s.conf.Stream,
		Domain:    "qn-ss.douyucdn.cn",
		Type:      "live",
		Node:      nodeId,
		ConnectId: "connId",
		User:      "",
		PlayUrl:   "http://124.236.43.202:22282/qn-ss.douyucdn.cn/live1/stream011.xs?wsSecret=208262e79b30d92b8187646fdc3a1729&wsTime=65ae654e",
	}
	bytes, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("req:", string(bytes))
	var resp model.PathQueryResponse
	addr := "http://10.34.146.62:6060/api/v1/pathquery"
	respData, err := util.GetWithBody(addr, string(bytes))
	if err != nil {
		s.logger.Error().Err(err).Msg("req pathquery err")
		return
	}
	if err := json.Unmarshal([]byte(respData), &resp); err != nil {
		log.Println(err)
		return
	}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("resp:", string(data))
}

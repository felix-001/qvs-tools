package miku

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/public/util"
	"os"

	"github.com/qbox/mikud-live/common/model"
	"github.com/rs/zerolog"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

func (m *Miku) Pathquery() {
	nodeId, _ := util.GetPcdnFromSchedAPI(m.conf)
	if nodeId == "" {
		logger.Info().Str("area", m.conf.Area).Str("isp", m.conf.Isp).Msg("get pcdn err")
		return
	}
	playUrl := fmt.Sprintf("http://%s/%s/%s.%s?wsSecret=208262e79b30d92b8187646fdc3a1729&wsTime=65ae654e&QiNiuTestTag=%s&QiNiuTime=%s",
		m.conf.Domain, m.conf.Bucket, m.conf.Stream, m.conf.Format, util.QiNiuTestTag, util.QiniuTime)
	req := model.PathQueryRequest{
		Bucket:    m.conf.Bucket,
		Key:       m.conf.Stream,
		Domain:    m.conf.Domain,
		Type:      "live",
		Node:      nodeId,
		ConnectId: m.conf.ConnId,
		User:      m.conf.User,
		PlayUrl:   playUrl,
	}
	bytes, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("req:", string(bytes))
	var resp model.PathQueryResponse
	addr := fmt.Sprintf("http://%s:6060/api/v1/pathquery", m.conf.SchedIp)
	respData, err := util.GetWithBody(addr, string(bytes))
	if err != nil {
		logger.Error().Err(err).Msg("req pathquery err")
		return
	}
	if err = json.Unmarshal([]byte(respData), &resp); err != nil {
		log.Println(err)
		return
	}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("resp:", string(data))
}

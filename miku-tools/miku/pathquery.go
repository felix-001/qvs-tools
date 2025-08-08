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
	playUrl := fmt.Sprintf("http://%s/%s/%s.%s?wsSecret=208262e79b30d92b8187646fdc3a1729&wsTime=65ae654e",
		m.conf.Domain, m.conf.Bucket, m.conf.Stream, m.conf.Format)
	req := model.PathQueryRequest{
		Bucket:    m.conf.Bucket,
		Key:       m.conf.Stream,
		Domain:    m.conf.Domain,
		Type:      "live",
		Node:      nodeId,
		ConnectId: m.conf.ConnId,
		User:      m.conf.User,
		PlayUrl:   playUrl,
		OriginUrl: m.conf.Origin,
	}
	bytes, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("req:", string(bytes))
	var resp model.PathQueryResponse
	addr := fmt.Sprintf("http://%s:6060/api/v1/pathquery?QiNiuTestTag=%s&QiNiuTime=%s", m.conf.SchedIp, util.QiNiuTestTag, util.QiniuTime)
	fmt.Println("addr:", addr)
	clientIp := m.resources.V4Ips[m.conf.Isp][m.conf.Province]
	if clientIp == "" {
		logger.Error().Str("isp", m.conf.Isp).Str("prov", m.conf.Province).Msg("get ip err")
		return
	}
	headers := map[string]string{
		"X-Real-IP": clientIp,
	}
	respData, err := util.HttpReq("GET", addr, string(bytes), headers)
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

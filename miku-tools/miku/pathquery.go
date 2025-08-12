package miku

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/public/util"
	"os"
	"strconv"
	"strings"

	"github.com/qbox/mikud-live/common/model"
	"github.com/rs/zerolog"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

func (m *Miku) Pathquery() {
	node := m.conf.Node
	pcdn := ""
	if !m.conf.Local {
		nodeId, pcdnId := util.GetPcdnFromSchedAPI(m.conf)
		if nodeId == "" {
			logger.Info().Str("area", m.conf.Area).Str("isp", m.conf.Isp).Msg("get pcdn err")
			return
		}
		pcdn = pcdnId
		if node == "" {
			node = nodeId
		}
	}
	clientIp := m.conf.Ip
	if pcdn != "" {
		fields := strings.Split(pcdn, ":")
		if len(fields) != 2 {
			logger.Error().Str("pcdn", pcdn).Msg("pcdn format err")
			return
		}
		clientIp = fields[0]
	}
	playUrl := fmt.Sprintf("http://%s/%s/%s.%s?wsSecret=208262e79b30d92b8187646fdc3a1729&wsTime=65ae654e",
		m.conf.Domain, m.conf.Bucket, m.conf.Stream, m.conf.Format)
	if m.conf.Format == "slice" {
		res, err := util.InetAton(m.conf.Origin)
		if err != nil {
			logger.Error().Str("origin", m.conf.Origin).Msg("inet aton err")
			return
		}
		playUrl += "&ex1=" + strconv.FormatInt(int64(res), 10)
	}
	req := model.PathQueryRequest{
		Bucket:    m.conf.Bucket,
		Key:       m.conf.Stream,
		Domain:    m.conf.Domain,
		Type:      "live",
		Node:      node,
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

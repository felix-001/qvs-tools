package miku

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"mikutool/public/util"
	"os"
	"strconv"
	"strings"

	"github.com/qbox/mikud-live/common/model"
	"github.com/rs/zerolog"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

type PathQueryResponse struct {
	Code    string              `json:"code"`    // 业务错误码
	Message string              `json:"message"` // 错误码对应的可读性解释词
	Sources []*model.SourceItem `json:"sources"` // 完成的调度路径，格式为回源url
	//Path    []*PathItemInfo `json:"path"`    // 完整的调度路径，包括推流点、内部转发路径、拉流点
	//StreamConf StreamConfig    `json:"streamConf"`
	ConnectId string `json:"connectId"`
	Ttl       int    `json:"ttl"` // 等待时间并重试，单位s（由 agent 处理）

	// fixme：过渡控制器，后续 miku 完全独立计量后去掉
	FlowMethod int `json:"flowMethod"` // 计量方式: 1: miku计量系统; 2: pili计量系统; 其它值miku&pili计量系统
}

func (m *Miku) Pathquery() {
	node := m.conf.Node
	pcdn := ""
	if !m.conf.Local {
		nodeId, pcdnId := util.GetPcdnFromSchedAPI(m.conf)
		if nodeId == "" {
			logger.Info().Str("area", m.conf.Area).Str("isp", m.conf.Isp).Msg("get pcdn err")
			nodeId, pcdnId = util.GetRandomPcdnFromSchedAPI(m.conf)
			if nodeId == "" {
				logger.Info().Str("isp", m.conf.Isp).Msg("get random pcdn err")
				return
			}
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
	if m.conf.App == "" {
		m.conf.App = m.conf.Bucket
	}
	playUrl := fmt.Sprintf("http://%s/%s/%s.%s?wsSecret=208262e79b30d92b8187646fdc3a1729&wsTime=65ae654e",
		m.conf.Domain, m.conf.App, m.conf.Stream, m.conf.Format)
	if m.conf.QnTestUrl != "" {
		playUrl += "&qnTestUrl=" + m.conf.QnTestUrl
	}
	if m.conf.RawApp != "" {
		playUrl += "&rawApp=" + m.conf.RawApp
	}
	if m.conf.Redirect {
		playUrl = fmt.Sprintf("http://127.0.0.1/%s/%s/%s.%s?wsSecret=208262e79b30d92b8187646fdc3a1729&wsTime=65ae654e&domain=%s",
			m.conf.Domain, m.conf.Bucket, m.conf.Stream, m.conf.Format, m.conf.Domain)

	}
	if m.conf.Format == "slice" {
		res, err := util.InetAton(m.conf.Origin)
		if err != nil {
			logger.Error().Str("origin", m.conf.Origin).Msg("inet aton err")
			return
		}
		playUrl += "&ex1=" + strconv.FormatInt(int64(res), 10)
	}
	// 生成10字节随机字符串作为ConnId
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	m.conf.ConnId = string(b)
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
		Skip:      strings.Split(m.conf.Skip, ","),
	}
	bytes, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("req:", string(bytes))
	var resp PathQueryResponse
	port := m.conf.Port
	if port == 0 {
		port = 6060
	}
	addr := fmt.Sprintf("http://%s:%d/api/v1/pathquery?QiNiuTestTag=%s&QiNiuTime=%s", m.conf.SchedIp, port, util.QiNiuTestTag, util.QiniuTime)
	fmt.Println("addr:", addr)

	headers := map[string]string{
		"X-Real-IP": clientIp,
	}
	respData, err := util.HttpReq("GET", addr, string(bytes), headers, m.conf.Detail)
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

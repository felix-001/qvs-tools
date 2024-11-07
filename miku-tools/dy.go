package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (s *Parser) dySecret() (string, string) {
	t, err := str2time("2034-12-01 00:00:00")
	if err != nil {
		s.logger.Error().Err(err).Msg("conv time err")
		return "", ""
	}
	wsTime := fmt.Sprintf("%x", t.Unix())
	raw := s.conf.Secret + s.conf.Stream + wsTime
	hash := md5.Sum([]byte(raw))
	wsSecret := hex.EncodeToString([]byte(hash[:]))
	return wsTime, wsSecret
}

func (s *Parser) DyPlay() {
	//pcdn := s.getPcdn()
	if s.conf.Pcdn == "" {
		_, s.conf.Pcdn = s.getPcdnFromSchedAPI(true, false)
	}
	wsTime, wsSecret := s.dySecret()
	cmdStr := fmt.Sprintf("./xs -addr %s -path %s/%s.xs -q \"origin=tct&wsSecret=%s&wsTime=%s&domain=%s&sourceID=%s\" -f out.xs",
		s.conf.Pcdn, s.conf.Bucket, s.conf.Stream, wsSecret, wsTime, s.conf.Domain, s.conf.SourceId)
	log.Println("cmd:", cmdStr)
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v %s %s\n", err, string(output), cmdStr)
		return
	}
	fmt.Println(string(output))

}

func (s *Parser) DyPcdn() {
	fmt.Println(s.getPcdnFromSchedAPI(true, false))
}

type MetricResp struct {
	Code int `json:"code"`
	Data struct {
		Metrics struct {
			PushTimeout map[string]int `json:"push_timeout"`
		} `json:"metrics"`
	} `json:"data"`
}

func (s *Parser) getMetrics(t int64) *MetricResp {
	ts := fmt.Sprintf("%x", t)
	wsTime := fmt.Sprintf("%x", time.Now().Unix())
	seed := s.conf.DyApiSecret + "qiniu" + wsTime
	hash := md5.Sum([]byte(seed))
	wsSecret := hex.EncodeToString(hash[:])
	addr := fmt.Sprintf("http://%s/pcdn/v1/metrics/top_nodes/qiniu/?timestamp=%s&topn=20&wsSecret=%s&wsTime=%s",
		s.conf.DyApiDomain, ts, wsSecret, wsTime)
	metrics, err := get(addr)
	if err != nil {
		s.logger.Error().Err(err).Str("addr", addr).Msg("req dy metrics err")
		return nil
	}
	fmt.Println(metrics)
	var resp MetricResp
	if err := json.Unmarshal([]byte(metrics), &resp); err != nil {
		log.Println(err)
		return nil
	}
	return &resp
}

func (s *Parser) GetDyMetrics() {
	t, err := str2unix(s.conf.T)
	if err != nil {
		s.logger.Error().Err(err).Msg("str2unix err")
		return
	}
	resp := s.getMetrics(t)
	//fmt.Println(resp)
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

func (s *Parser) GetDyTimeout() {
	t, err := str2unix(s.conf.T)
	if err != nil {
		s.logger.Error().Err(err).Msg("str2unix err")
		return
	}
	nodeCntMap := make(map[string]int)
	// 5分钟查询一次
	for i := 0; i < 24*60/5; i++ {
		log.Println(i)
		resp := s.getMetrics(t)
		for pcdnId := range resp.Data.Metrics.PushTimeout {
			parts := strings.Split(pcdnId, "/")
			if len(parts) != 3 {
				s.logger.Error().Str("pcdnId", pcdnId).Msg("parse pcdnId err")
				continue
			}
			nodeCntMap[parts[0]]++
		}
		t += 5 * 60
		time.Sleep(time.Second)
	}
	pairs := SortIntMap(nodeCntMap)
	for _, pair := range pairs {
		machineId := ""
		node := s.allNodesMap[pair.Key]
		if node != nil {
			machineId = node.MachineId
		}
		s.logger.Info().Str("nodeId", pair.Key).Int("cnt", pair.Value).Str("machineId", machineId).Msg("")

	}
}

func (s *Parser) DyOriginal() {
	app := s.conf.Bucket
	stream := s.conf.Stream
	key := s.conf.OriginKey
	domain := s.conf.Domain

	expireTime := time.Now().Unix() + int64(600)
	hexTime := strconv.FormatInt(expireTime, 16)
	raw := fmt.Sprintf("%s%s%s", key, stream, hexTime)
	hash := md5.Sum([]byte(raw))
	txSecret := hex.EncodeToString([]byte(hash[:]))
	originUrl := fmt.Sprintf("http://%s/%s/%s.flv?txSecret=%s&txTime=%s", domain,
		app, stream, txSecret, hexTime)
	fmt.Println("url: ", originUrl)
}

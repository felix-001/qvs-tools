package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"middle-source-analysis/config"
	"middle-source-analysis/util"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm/logger"
)

var (
	Conf *config.Config
)

func dySecret() (string, string) {
	t, err := str2time("2034-12-01 00:00:00")
	if err != nil {
		logger.Error().Err(err).Msg("conv time err")
		return "", ""
	}
	wsTime := fmt.Sprintf("%x", t.Unix())
	raw := Conf.Secret + Conf.Stream + wsTime
	hash := md5.Sum([]byte(raw))
	wsSecret := hex.EncodeToString([]byte(hash[:]))
	return wsTime, wsSecret
}

func DyPlay() {
	//pcdn := getPcdn()
	if Conf.Pcdn == "" {
		_, Conf.Pcdn = getPcdnFromSchedAPI(true, false)
	}
	wsTime, wsSecret := dySecret()
	cmdStr := fmt.Sprintf("./xs -addr %s -path %s/%xs -q \"origin=tct&wsSecret=%s&wsTime=%s&domain=%s&sourceID=%s\" -f out.xs",
		Conf.Pcdn, Conf.Bucket, Conf.Stream, wsSecret, wsTime, Conf.Domain, Conf.SourceId)
	log.Println("cmd:", cmdStr)
	cmd := exec.Command("bash", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v %s %s\n", err, string(output), cmdStr)
		return
	}
	fmt.Println(string(output))

}

func DyPcdn() {
	fmt.Println(getPcdnFromSchedAPI(true, false))
}

type MetricResp struct {
	Code int `json:"code"`
	Data struct {
		Metrics struct {
			PushTimeout map[string]int `json:"push_timeout"`
		} `json:"metrics"`
	} `json:"data"`
}

func getMetrics(t int64) *MetricResp {
	ts := fmt.Sprintf("%x", t)
	wsTime := fmt.Sprintf("%x", time.Now().Unix())
	seed := Conf.DyApiSecret + "qiniu" + wsTime
	hash := md5.Sum([]byte(seed))
	wsSecret := hex.EncodeToString(hash[:])
	addr := fmt.Sprintf("http://%s/pcdn/v1/metrics/top_nodes/qiniu/?timestamp=%s&topn=20&wsSecret=%s&wsTime=%s",
		Conf.DyApiDomain, ts, wsSecret, wsTime)
	metrics, err := util.Get(addr)
	if err != nil {
		logger.Error().Err(err).Str("addr", addr).Msg("req dy metrics err")
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

func GetDyMetrics() {
	t, err := str2unix(Conf.T)
	if err != nil {
		logger.Error().Err(err).Msg("str2unix err")
		return
	}
	resp := getMetrics(t)
	//fmt.Println(resp)
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

func GetDyTimeout() {
	t, err := str2unix(Conf.T)
	if err != nil {
		logger.Error().Err(err).Msg("str2unix err")
		return
	}
	nodeCntMap := make(map[string]int)
	// 5分钟查询一次
	for i := 0; i < 24*60/5; i++ {
		log.Println(i)
		resp := getMetrics(t)
		for pcdnId := range resp.Data.MetricPushTimeout {
			parts := strings.Split(pcdnId, "/")
			if len(parts) != 3 {
				logger.Error().Str("pcdnId", pcdnId).Msg("parse pcdnId err")
				continue
			}
			nodeCntMap[parts[0]]++
		}
		t += 5 * 60
		time.Sleep(time.Second)
	}
	pairs := util.SortIntMap(nodeCntMap)
	for _, pair := range pairs {
		machineId := ""
		node := allNodesMap[pair.Key]
		if node != nil {
			machineId = node.MachineId
		}
		logger.Info().Str("nodeId", pair.Key).Int("cnt", pair.Value).Str("machineId", machineId).Msg("")

	}
}

func DyOriginal() {
	app := Conf.Bucket
	stream := Conf.Stream
	key := Conf.OriginKey
	domain := Conf.Domain

	if Conf.Origin == "dy" {
		key = Conf.OriginKeyDy
	} else if Conf.Origin == "hw" {
		key = Conf.OriginKeyHw
	}

	expireTime := time.Now().Unix() + int64(600)
	hexTime := strconv.FormatInt(expireTime, 16)
	raw := fmt.Sprintf("%s%s%s", key, stream, hexTime)
	hash := md5.Sum([]byte(raw))
	txSecret := hex.EncodeToString([]byte(hash[:]))
	originUrl := fmt.Sprintf("http://%s/%s/%flv?txSecret=%s&txTime=%s&origin=%s", domain,
		app, stream, txSecret, hexTime, Conf.Origin)
	fmt.Println("url: ", originUrl)
}

type XsPlayCmd struct {
	Cmd       string `json:"cmd"`
	Basesub   int    `json:"basesub"`
	Startid   int    `json:"startid"`
	Substream int    `json:"substream"`
}

func XsPlay() {
	if Conf.Pcdn == "" {
		_, Conf.Pcdn = getPcdnFromSchedAPI(true, false)
	}
	wsTime, wsSecret := dySecret()
	addr := fmt.Sprintf("ws://%s/%s/%s/%xs?wsSecret=%s&origin=%s&wsTime=%s&sourceID=%s",
		Conf.Pcdn, Conf.Domain, Conf.Bucket, Conf.Stream, wsSecret, Conf.Origin,
		wsTime, Conf.SourceId)
	logger.Info().Str("addr", addr).Msg("XsPlay")
	c, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer c.Close()

	cmd := &XsPlayCmd{
		Cmd:       "play",
		Basesub:   Conf.Basesub,
		Substream: Conf.SubStream,
		Startid:   Conf.Startid,
	}
	bytes, err := json.Marshal(cmd)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("send cmd:", string(bytes))
	if err := c.WriteMessage(websocket.TextMessage, bytes); err != nil {
		fmt.Println("write:", err)
		return
	}
	file, err := os.OpenFile(Conf.F, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	total := 0
	defer func() {
		logger.Info().Int("totalRecv", total).Msg("")
	}()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			logger.Error().Err(err).Msg("ws read")
			return
		}
		_, err = file.Write(message)
		if err != nil {
			panic(err)
		}
		total += len(message)
	}
}

type mongoTime struct {
	Date time.Time `json:"$date"`
}

type ForbiddenNode2 struct {
	Ts       mongoTime `json:"ts"`
	OutBw    float64   `json:"outBw"`
	MaxOutBw float64   `json:"maxOutBw"`
	Overflow bool      `json:"overflow"`
}

type DyAbnormalNodesInfo struct {
	Metrics struct {
		PushTimeout map[string]int `json:"push_timeout" bson:"push_timeout"`
		ConnectFail map[string]int `json:"connect_fail" bson:"connect_fail"`
	} `json:"metrics" bson:"metrics"`
	TotalTimeoutNodes       int                       `json:"total_timeout_nodes" bson:"total_timeout_nodes"`
	TimeoutForbiddenNodes   map[string]ForbiddenNode2 `json:"timeout_forbidden_nodes" bson:"timeout_forbidden_nodes"`
	TotalErrNodes           int                       `json:"total_err_nodes" bson:"total_err_nodes"`
	PcdnErrFbiddenNodes     map[string]ForbiddenNode2 `json:"pcdn_err_forbidden_nodes" bson:"pcdn_err_forbidden_nodes"`
	TotalConnectFailNodes   int                       `json:"total_connect_fail_nodes" bson:"total_connect_fail_nodes"`
	ConnectFailFbiddenNodes map[string]ForbiddenNode2 `json:"connect_fail_forbidden_nodes" bson:"connect_fail_forbidden_nodes"`
	CreatedAt               mongoTime                 `json:"created_at" bson:"created_at"`
	NodeId                  string                    `json:"node_id" bson:"node_id"`
}

func PushTimeout() {
	bytes, err := os.ReadFile("/Users/liyuanquan/workspace/tmp/douyu-blacklist-2025-0303-1600.json")
	if err != nil {
		log.Println("read fail", "/Users/liyuanquan/workspace/tmp/export_test.json", err)
		return
	}

	metrics := make([]DyAbnormalNodesInfo, 0)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var metric DyAbnormalNodesInfo
		if err := json.Unmarshal([]byte(line), &metric); err != nil {
			log.Println("unmarshal err 1", err, line)
			continue
		}
		metrics = append(metrics, metric)
	}
	log.Println(len(metrics))
	traceNode(metrics)
	//dumpNodeMap(metrics)
}

func dumpNodeMap(metrics []DyAbnormalNodesInfo) {
	nodeMap := make(map[string]int)
	for _, metric := range metrics {
		for nodeId := range metric.MetricPushTimeout {
			nodeMap[nodeId]++
		}
	}
	pairs := util.SortIntMap(nodeMap)
	util.DumpSlice(pairs)
}

func traceNode(metrics []DyAbnormalNodesInfo) {
	/*
		loc, err := time.LoadLocation("Asia/Shanghai") // 加载东八区时区[1](@ref)
		if err != nil {
			panic(err)
		}
	*/
	for _, metric := range metrics {
		//fmt.Println("createdAt:", metric.CreatedAt.Date.In(loc))
		for pcdnId, detail := range metric.TimeoutForbiddenNodes {
			if strings.Contains(pcdnId, "593d5a07-698b-3fe5-9e12-e2decf427400-niulink64-site") {
				fmt.Println(pcdnId, detail)
			}
		}
		for pcdnId, detail := range metric.PcdnErrFbiddenNodes {
			if strings.Contains(pcdnId, "593d5a07-698b-3fe5-9e12-e2decf427400-niulink64-site") {
				fmt.Println(pcdnId, detail)
			}
		}

	}
}

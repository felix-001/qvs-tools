package miku

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"mikutool/public/util"

	commonModel "github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
)

const defaultOriginNodeAPI = "http://miku-lived.qiniuapi.com:2240/v1/runtime/nodes"

type originRecord struct {
	NodeId   string           `json:"nodeId"`
	StreamId string           `json:"streamId"`
	SQL      string           `json:"sql"`
	Results  []map[string]any `json:"results"`
}

func (m *Miku) Origin() {
	conf := m.conf
	if conf.F == "" {
		log.Println("缺少节点列表文件，请使用 -f <node_file>")
		return
	}
	if conf.T == "" {
		log.Println("缺少流列表文件，请使用 -t <stream_file>")
		return
	}

	nodeIds, err := readFirstColumn(conf.F)
	if err != nil {
		log.Printf("读取节点列表失败: %v", err)
		return
	}
	streamIds, err := readFirstColumn(conf.T)
	if err != nil {
		log.Printf("读取流列表失败: %v", err)
		return
	}
	log.Printf("节点数: %d, 流数: %d", len(nodeIds), len(streamIds))

	day := conf.Day
	if day == "" {
		day = "20260701"
	}
	hour := conf.Hour
	if hour == "" {
		hour = "14"
	}
	appname := conf.App
	if appname == "" {
		appname = "maozhua"
	}
	outputFile := conf.Output
	if outputFile == "" {
		outputFile = fmt.Sprintf("origin_result_%d.jsonl", time.Now().Unix())
	}

	out, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("打开输出文件失败: %v", err)
		return
	}
	defer out.Close()

	apiBase := conf.Addr
	if apiBase == "" {
		apiBase = defaultOriginNodeAPI
	}

	hitCnt := 0
	for i, nodeId := range nodeIds {
		nodeId = strings.TrimSpace(nodeId)
		if nodeId == "" {
			continue
		}
		log.Printf("[%d/%d] 查询节点: %s", i+1, len(nodeIds), nodeId)

		node, err := fetchRtNode(apiBase, nodeId)
		if err != nil {
			log.Printf("获取节点 %s 失败: %v", nodeId, err)
			continue
		}
		ips := getPublicIPs(node)
		if len(ips) == 0 {
			log.Printf("节点 %s 无公网 IP，跳过", nodeId)
			continue
		}
		log.Printf("节点 %s 公网 IP: %v", nodeId, ips)

		for j, streamId := range streamIds {
			streamId = strings.TrimSpace(streamId)
			if streamId == "" {
				continue
			}
			sql := buildOriginSQL(day, hour, appname, streamId, ips)
			var results []map[string]any
			if err := util.TrinoQueryMap("miku", sql, &results); err != nil {
				log.Printf("Trino 查询失败 node=%s stream=%s: %v", nodeId, streamId, err)
				continue
			}
			if len(results) == 0 {
				continue
			}

			record := originRecord{
				NodeId:   nodeId,
				StreamId: streamId,
				SQL:      sql,
				Results:  results,
			}
			data, err := json.Marshal(record)
			if err != nil {
				log.Printf("序列化结果失败: %v", err)
				continue
			}
			if _, err := out.Write(append(data, '\n')); err != nil {
				log.Printf("写入输出文件失败: %v", err)
				return
			}
			hitCnt++
			log.Printf("  [%d/%d] 命中 stream=%s, 结果数=%d", j+1, len(streamIds), streamId, len(results))
		}
	}
	log.Printf("完成，共命中 %d 条记录，输出文件: %s", hitCnt, outputFile)
}

func readFirstColumn(file string) ([]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var ids []string
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 && (strings.Contains(line, "nodeid") || strings.Contains(line, "streamname")) {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		ids = append(ids, parts[0])
	}
	return ids, nil
}

func fetchRtNode(apiBase, nodeId string) (*commonModel.RtNode, error) {
	reqURL := fmt.Sprintf("%s?fuzzy=%s", apiBase, url.QueryEscape(nodeId))
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var nodes []*commonModel.RtNode
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	for _, n := range nodes {
		if n.Id == nodeId {
			return n, nil
		}
	}
	if len(nodes) > 0 {
		return nodes[0], nil
	}
	return nil, fmt.Errorf("未找到节点")
}

func getPublicIPs(node *commonModel.RtNode) []string {
	seen := make(map[string]bool)
	var ips []string
	for _, ipInfo := range node.Ips {
		if publicUtil.IsPrivateIP(ipInfo.Ip) {
			continue
		}
		if seen[ipInfo.Ip] {
			continue
		}
		seen[ipInfo.Ip] = true
		ips = append(ips, ipInfo.Ip)
	}
	return ips
}

func buildOriginSQL(day, hour, appname, streamname string, ips []string) string {
	var ipConds []string
	for _, ip := range ips {
		ipConds = append(ipConds,
			fmt.Sprintf("localaddr LIKE '%%%s%%'", ip),
			fmt.Sprintf("remoteaddr LIKE '%%%s%%'", ip),
		)
	}
	ipClause := strings.Join(ipConds, " OR ")
	return fmt.Sprintf(`SELECT
   DISTINCT localaddr,remoteaddr, nodeid, request
FROM miku.dwd_flowd_miku_streamd_log
WHERE day = '%s'
  AND hour = '%s'
  AND type = 'internal-player'
  AND (%s)
  AND appname = '%s'
  AND streamname = '%s'`, day, hour, ipClause, appname, streamname)
}

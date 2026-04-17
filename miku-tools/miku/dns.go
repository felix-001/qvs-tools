package miku

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/qbox/pili/base/qiniu/xlog.v1"
)

func (m *Miku) DumpDns() {
	if m.conf.Domain == "" {
		log.Println("domain is empty")
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := m.resources.DnsPodCli.GetRecords(xl, m.conf.Domain, m.conf.Host, "", 0)
	if err != nil {
		log.Println(err)
		return
	}
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}

	// 转换为 CSV 并写入文件
	if err := m.convertToCSV(string(bytes)); err != nil {
		log.Printf("转换为 CSV 失败: %v", err)
		return
	}

	fmt.Println(string(bytes))
}

func (m *Miku) convertToCSV(jsonStr string) error {
	// 解析 JSON 响应
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 提取记录列表
	var records []map[string]interface{}
	if data["records"] != nil {
		if recs, ok := data["records"].([]interface{}); ok {
			for _, rec := range recs {
				if recMap, ok := rec.(map[string]interface{}); ok {
					records = append(records, recMap)
				}
			}
		}
	} else if data["RecordList"] != nil {
		// 尝试另一种可能的结构
		if recs, ok := data["RecordList"].([]interface{}); ok {
			for _, rec := range recs {
				if recMap, ok := rec.(map[string]interface{}); ok {
					records = append(records, recMap)
				}
			}
		}
	} else if data["data"] != nil {
		// 尝试标准 API 响应结构
		if dataMap, ok := data["data"].(map[string]interface{}); ok {
			if recs, ok := dataMap["records"].([]interface{}); ok {
				for _, rec := range recs {
					if recMap, ok := rec.(map[string]interface{}); ok {
						records = append(records, recMap)
					}
				}
			}
		}
	}

	if len(records) == 0 {
		return fmt.Errorf("未找到 DNS 记录")
	}

	// 生成 CSV 文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("dns_records_%s_%s.csv", m.conf.Domain, timestamp)
	filepath := filepath.Join("/tmp", filename)

	// 创建 CSV 文件
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 创建 CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{"ID", "Host", "Type", "Value", "TTL", "Status", "Domain", "Line"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("写入表头失败: %w", err)
	}

	// 写入记录
	for _, record := range records {
		row := []string{
			fmt.Sprintf("%v", getValue(record, "id", "ID")),
			fmt.Sprintf("%v", getValue(record, "host", "Host", "name")),
			fmt.Sprintf("%v", getValue(record, "type", "Type")),
			fmt.Sprintf("%v", getValue(record, "value", "Value")),
			fmt.Sprintf("%v", getValue(record, "ttl", "TTL")),
			fmt.Sprintf("%v", getValue(record, "status", "Status")),
			fmt.Sprintf("%v", getValue(record, "domain", "Domain")),
			fmt.Sprintf("%v", getValue(record, "line", "Line")),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入记录失败: %w", err)
		}
	}

	log.Printf("DNS 记录已保存到: %s", filepath)
	return nil
}

// getValue 从 map 中获取值，支持多个键名
func getValue(m map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			return val
		}
	}
	return ""
}

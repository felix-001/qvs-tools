package miku

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/qbox/mikud-live/cmd/dnspod/model"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
)

type RecordListItem struct {
	// 记录Id
	RecordId *uint64 `json:"RecordId,omitempty" name:"RecordId"`

	// 记录值
	Value *string `json:"Value,omitempty" name:"Value"`

	// 记录状态，启用：ENABLE，暂停：DISABLE
	Status *string `json:"Status,omitempty" name:"Status"`

	// 更新时间
	UpdatedOn *string `json:"UpdatedOn,omitempty" name:"UpdatedOn"`

	// 主机名
	Name *string `json:"Name,omitempty" name:"Name"`

	// 记录线路
	Line *string `json:"Line,omitempty" name:"Line"`

	// 线路Id
	LineId *string `json:"LineId,omitempty" name:"LineId"`

	// 记录类型
	Type *string `json:"Type,omitempty" name:"Type"`

	// 记录权重，用于负载均衡记录
	// 注意：此字段可能返回 null，表示取不到有效值。
	Weight *uint64 `json:"Weight,omitempty" name:"Weight"`

	// 记录监控状态，正常：OK，告警：WARN，宕机：DOWN，未设置监控或监控暂停则为空
	MonitorStatus *string `json:"MonitorStatus,omitempty" name:"MonitorStatus"`

	// 记录备注说明
	Remark *string `json:"Remark,omitempty" name:"Remark"`

	// 记录缓存时间
	TTL *uint64 `json:"TTL,omitempty" name:"TTL"`

	// MX值，只有MX记录有
	// 注意：此字段可能返回 null，表示取不到有效值。
	MX *uint64 `json:"MX,omitempty" name:"MX"`

	// 是否是默认的ns记录
	DefaultNS *bool `json:"DefaultNS,omitempty" name:"DefaultNS"`
}

func (m *Miku) Dns() {
	if m.conf.Domain == "" {
		log.Println("domain is empty")
		return
	}
	if m.conf.Del {
		log.Println("del dns record")
		m.DelDnsRecord()
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := m.resources.DnsPodCli.GetRecords(xl, m.conf.Domain, m.conf.Host, "", 0)
	if err != nil {
		log.Println(err)
		return
	}
	if m.conf.Enable {
		var recordListFiltered []RecordListItem
		for _, item := range resp.RecordList {
			if *item.Status != "ENABLE" {
				continue
			}
			if m.conf.Line != "" && *item.Line != m.conf.Line {
				continue
			}
			record := RecordListItem{
				RecordId:      item.RecordId,
				Value:         item.Value,
				Status:        item.Status,
				UpdatedOn:     item.UpdatedOn,
				Name:          item.Name,
				Line:          item.Line,
				LineId:        item.LineId,
				Type:          item.Type,
				Weight:        item.Weight,
				MonitorStatus: item.MonitorStatus,
				Remark:        item.Remark,
				TTL:           item.TTL,
				MX:            item.MX,
				DefaultNS:     item.DefaultNS,
			}
			recordListFiltered = append(recordListFiltered, record)
		}
		bytes, err := json.MarshalIndent(recordListFiltered, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(bytes))
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
	headers := []string{"ID", "Host", "Type", "Value", "TTL", "Status", "Domain", "Line", "Weight"}
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
			fmt.Sprintf("%v", getValue(record, "weight", "Weight")),
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

func (m *Miku) DelDnsRecord() {
	if m.conf.ID == "" {
		log.Println("id is empty")
		return
	}
	recordID, err := strconv.ParseUint(m.conf.ID, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	record := model.Record{
		Domain: m.conf.Domain,
		DnspodRecord: &model.DnspodRecord{
			RecordID: recordID,
		},
	}
	resp, err := m.resources.DnsPodCli.Delete(xl, record)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(resp)
}

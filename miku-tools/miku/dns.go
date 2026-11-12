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
	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
)

const (
	// dnsPageRetry 单页最大尝试次数(含首次)。超时多为服务端深分页抖动，重试可恢复
	dnsPageRetry = 3
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
	if resp == nil || len(resp.RecordList) == 0 {
		log.Println("未找到 DNS 记录")
		return
	}

	var writer *csv.Writer
	var csvPath, jsonPath string
	if !m.conf.Enable {
		timestamp := time.Now().Format("20060102_150405")
		csvPath = filepath.Join("/tmp", fmt.Sprintf("dns_records_%s_%s.csv", m.conf.Domain, timestamp))
		jsonPath = filepath.Join("/tmp", fmt.Sprintf("dns_records_%s_%s.json", m.conf.Domain, timestamp))
		csvFile, err := os.Create(csvPath)
		if err != nil {
			log.Printf("创建 CSV 失败: %v", err)
			return
		}
		defer csvFile.Close()
		writer = csv.NewWriter(csvFile)
		defer writer.Flush()
		if err := writer.Write(dnsCSVHeaders); err != nil {
			log.Printf("写入 CSV 表头失败: %v", err)
			return
		}
		if err := dumpDNSPage(writer, csvPath, jsonPath, resp, resp); err != nil {
			log.Printf("写入第 1 页失败: %v", err)
			return
		}
	}

	// 单次请求最多返回 MaxRecordListLimit 条，记录数超出时串行分页拉全量
	if resp.RecordCountInfo != nil && resp.RecordCountInfo.TotalCount != nil {
		total := *resp.RecordCountInfo.TotalCount
		limit := tencent_dnspod.MaxRecordListLimit
		if total > limit {
			pages := int((total + limit - 1) / limit)
			log.Printf("域名 %s 共 %d 条记录，超过单页上限 %d，开始串行分页拉取，共 %d 页",
				m.conf.Domain, total, limit, pages)

			start := time.Now()
			for page := 1; page < pages; page++ {
				offset := limit * uint64(page)
				log.Printf("分页进度 %d/%d offset=%d 已获取 %d/%d 条",
					page+1, pages, offset, len(resp.RecordList), total)

				result, err := m.getRecordsWithRetry(xl.Spawn(), offset)
				if err != nil {
					log.Printf("分页拉取失败 page=%d offset=%d: %v，已获取 %d/%d 条，部分结果已写入 csv/json",
						page+1, offset, err, len(resp.RecordList), total)
					return
				}
				if result != nil {
					resp.RecordList = append(resp.RecordList, result.RecordList...)
					if writer != nil {
						if err := dumpDNSPage(writer, csvPath, jsonPath, result, resp); err != nil {
							log.Printf("写入第 %d 页失败: %v", page+1, err)
							return
						}
					}
				}
			}
			log.Printf("分页拉取完成，实际获取 %d/%d 条，耗时 %s",
				len(resp.RecordList), total, time.Since(start).Round(time.Second))
		}
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
}

func (m *Miku) getRecordsWithRetry(xl *xlog.Logger, offset uint64) (*model.DnspodRecordsResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= dnsPageRetry; attempt++ {
		resp, err := m.resources.DnsPodCli.GetRecords(xl, m.conf.Domain, m.conf.Host, "", offset)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < dnsPageRetry {
			log.Printf("分页重试 offset=%d 第 %d 次失败: %v", offset, attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, lastErr
}

var dnsCSVHeaders = []string{
	"RecordId", "Value", "Status", "UpdatedOn", "Name",
	"Line", "LineId", "Type", "Weight", "MonitorStatus",
	"Remark", "TTL", "MX", "DefaultNS",
}

// dumpDNSPage 将本页记录追加到 CSV，并用累计结果覆盖 JSON
func dumpDNSPage(writer *csv.Writer, csvPath, jsonPath string, page, all *model.DnspodRecordsResponse) error {
	if err := appendCSVRecords(writer, page); err != nil {
		return err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("写入 CSV 失败: %w", err)
	}
	if err := writeDNSJSON(jsonPath, all); err != nil {
		return err
	}
	log.Printf("已更新文件 csv=%s json=%s 累计 %d 条", csvPath, jsonPath, len(all.RecordList))
	return nil
}

func appendCSVRecords(writer *csv.Writer, resp *model.DnspodRecordsResponse) error {
	if resp == nil {
		return nil
	}
	for _, record := range resp.RecordList {
		if record == nil {
			continue
		}
		row := []string{
			formatUint64(record.RecordId),
			formatString(record.Value),
			formatString(record.Status),
			formatString(record.UpdatedOn),
			formatString(record.Name),
			formatString(record.Line),
			formatString(record.LineId),
			formatString(record.Type),
			formatUint64(record.Weight),
			formatString(record.MonitorStatus),
			formatString(record.Remark),
			formatUint64(record.TTL),
			formatUint64(record.MX),
			formatBool(record.DefaultNS),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入记录失败: %w", err)
		}
	}
	return nil
}

func writeDNSJSON(path string, resp *model.DnspodRecordsResponse) error {
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("写入 JSON 失败: %w", err)
	}
	return nil
}

// formatString 将可能为 nil 的字符串指针转为 CSV 字段值
func formatString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// formatUint64 将可能为 nil 的 uint64 指针转为 CSV 字段值
func formatUint64(v *uint64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatUint(*v, 10)
}

// formatBool 将可能为 nil 的 bool 指针转为 CSV 字段值
func formatBool(v *bool) string {
	if v == nil {
		return ""
	}
	return strconv.FormatBool(*v)
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

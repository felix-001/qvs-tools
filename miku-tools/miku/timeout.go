package miku

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
)

type hostCount struct {
	host  string
	count int
}

// TimeoutStat 读取 CSV 文件，按 host 字段聚合统计条目数，从大到小输出
func (m *Miku) TimeoutStat(csvPath string) {
	f, err := os.Open(csvPath)
	if err != nil {
		log.Fatalf("[TimeoutStat] 打开文件失败: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// 读取并解析 header 行
	header, err := r.Read()
	if err != nil {
		log.Fatalf("[TimeoutStat] 读取 header 失败: %v", err)
	}

	hostIdx := -1
	for i, col := range header {
		log.Println(i, col)
		if col == "host" {
			hostIdx = i
			break
		}
	}
	if hostIdx < 0 {
		log.Fatalf("[TimeoutStat] CSV 中未找到 host 列，header: %v", header)
	}

	counts := make(map[string]int)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("[TimeoutStat] 读取行出错（跳过）: %v", err)
			continue
		}
		if hostIdx >= len(row) {
			continue
		}
		counts[row[hostIdx]]++
	}

	// 排序：按 count 从大到小
	result := make([]hostCount, 0, len(counts))
	for host, cnt := range counts {
		result = append(result, hostCount{host: host, count: cnt})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].count != result[j].count {
			return result[i].count > result[j].count
		}
		return result[i].host < result[j].host
	})

	fmt.Printf("%-60s %s\n", "host", "count")
	fmt.Printf("%-60s %s\n", "----", "-----")
	for _, item := range result {
		fmt.Printf("%-60s %d\n", item.host, item.count)
	}
}

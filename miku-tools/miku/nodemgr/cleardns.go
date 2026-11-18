package nodemgr

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	commonModel "github.com/qbox/mikud-live/common/model"
)

const defaultClearDnsCSV = "/tmp/dns_records_cloudvdn.com_20261112_120902_alnum.csv"

func (m *NodeMgr) ClearDns() {
	input := defaultClearDnsCSV
	if m.conf.F != "" {
		input = m.conf.F
	}
	output := m.conf.Output
	if output == "" {
		output = filepath.Join("/tmp", fmt.Sprintf("cleardns_%s.csv", time.Now().Format("20060102_150405")))
	}

	ipToNode := make(map[string]*commonModel.RtNode)
	for _, node := range m.allNodesMap {
		if node == nil || !node.IsDynamic {
			continue
		}
		for _, ipInfo := range node.Ips {
			ip := strings.TrimSpace(ipInfo.Ip)
			if ip == "" {
				continue
			}
			ipToNode[ip] = node
		}
	}
	log.Println("cleardns dynamic ip map size:", len(ipToNode), "nodes:", len(m.allNodesMap))

	inFile, err := os.Open(input)
	if err != nil {
		log.Println("cleardns open input csv err:", err)
		return
	}
	defer inFile.Close()

	outFile, err := os.Create(output)
	if err != nil {
		log.Println("cleardns create output csv err:", err)
		return
	}
	defer outFile.Close()

	reader := csv.NewReader(inFile)
	reader.FieldsPerRecord = -1
	writer := csv.NewWriter(outFile)
	defer writer.Flush()
	if err := writer.Write([]string{"nodeId", "machineId", "name", "ip"}); err != nil {
		log.Println("cleardns write header err:", err)
		return
	}

	scanned := 0
	matched := 0
	first := true
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("cleardns read csv err:", err)
			return
		}
		if first {
			first = false
			if len(row) > 0 && row[0] == "RecordId" {
				continue
			}
		}
		if len(row) < 5 {
			continue
		}
		scanned++
		ip := strings.TrimSpace(row[1])
		if ip == "" {
			continue
		}
		node, ok := ipToNode[ip]
		if !ok {
			continue
		}
		name := strings.TrimSpace(row[4])
		if err := writer.Write([]string{node.Id, node.MachineId, name, ip}); err != nil {
			log.Println("cleardns write row err:", err)
			return
		}
		matched++
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Println("cleardns flush err:", err)
		return
	}
	log.Printf("cleardns done scanned=%d matched=%d output=%s", scanned, matched, output)
}

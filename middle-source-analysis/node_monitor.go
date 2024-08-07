package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/qbox/mikud-live/cmd/sched/dal"
	"github.com/qbox/mikud-live/common/model"
	publicUtil "github.com/qbox/mikud-live/common/util"
)

const (
	IpStatusBwLimit              = "BwLimit"
	IpStatusStreamProbeStateFail = "StreamProbeStateFail"
	IpStatusStreamProbeSpeedFail = "StreamProbeSpeedFail"
	IpStatusForbidden            = "IpForbidden"
	IpStatusOffline              = "Offline"
	IpStatusBanProv              = "BanProv"
	IpStatusNoPorts              = "noPorts"
)

type IpInfo struct {
	IP     string `json:"ip"`
	Status string `json:"status"`
}

type NodeInfo struct {
	NodeId          string `json:"nodeid"`
	MachindId       string `json:"machineid"`
	RuntimeStatus   string `json:"runtime_status"`
	StreamdPorts    bool   `json:"streamd_ports"`
	HaveAvailableIp bool   `json:"hava_available_ip"`
	//AvailableIpCnt int      `json:"available_ip_cnt"`
	ErrIps    []IpInfo `json:"err_ips"`
	TimeStamp string   `json:"timestamp"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
}

func noStreamdPorts(node *model.RtNode) bool {
	return node.StreamdPorts.Http <= 0 || node.StreamdPorts.Https <= 0 || node.StreamdPorts.Wt <= 0
}

func checkIp(ipInfo model.RtIpStatus) bool {
	if publicUtil.IsPrivateIP(ipInfo.Ip) {
		return false
	}
	if ipInfo.Ip == "" {
		return false
	}
	if ipInfo.IsIPv6 {
		return false
	}
	return true
}

func (s *Parser) buildNodeInfo(node *model.RtNode) *NodeInfo {
	nodeInfo := NodeInfo{
		RuntimeStatus: node.RuntimeStatus,
		StreamdPorts:  !noStreamdPorts(node),
		//TimeStamp:     time.Now().Format("2006-01-02 15:04:05"),
		StartTime: time.Now().Format("2006-01-02 15:04:05"),
		NodeId:    node.Id,
		MachindId: node.MachineId,
	}
	availabeIpCnt := 0
	for _, ipInfo := range node.Ips {
		if !checkIp(ipInfo) {
			continue
		}
		if ipInfo.IPStreamProbe.State != model.StreamProbeStateSuccess {
			nodeInfo.ErrIps = append(nodeInfo.ErrIps, IpInfo{
				IP:     ipInfo.Ip,
				Status: IpStatusStreamProbeStateFail,
			})
			continue
		}
		if ipInfo.IPStreamProbe.Speed < 12 && ipInfo.IPStreamProbe.MinSpeed < 10 {
			nodeInfo.ErrIps = append(nodeInfo.ErrIps, IpInfo{
				IP:     ipInfo.Ip,
				Status: IpStatusStreamProbeSpeedFail,
			})
			continue
		}
		if ipInfo.OutMBps >= ipInfo.MaxOutMBps*0.8 {
			nodeInfo.ErrIps = append(nodeInfo.ErrIps, IpInfo{
				IP:     ipInfo.Ip,
				Status: IpStatusBwLimit,
			})
			continue
		}
		if ipInfo.Forbidden {
			nodeInfo.ErrIps = append(nodeInfo.ErrIps, IpInfo{
				IP:     ipInfo.Ip,
				Status: IpStatusForbidden,
			})
			continue
		}
		availabeIpCnt++
	}
	nodeInfo.HaveAvailableIp = (availabeIpCnt > 0)
	return &nodeInfo
}

func createFileName() string {
	timestamp := time.Now().Format("20060102150405") // 年月日时分秒
	return fmt.Sprintf("nodeinfo-%s.json", timestamp)
}

func deleteOldFiles() error {
	cutoff := time.Now().Add(-3 * 24 * time.Hour) // 3天前的日期
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.ModTime().Before(cutoff) {
			err := os.Remove(filepath.Join(path, file.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var path = "./node_info"

func (s *Parser) writeToFile(nodeInfo *NodeInfo) {
	createDirIfNotExist(path)
	if s.file == nil {
		fileName := createFileName()
		filePath := filepath.Join(path, fileName)
		var err error
		s.file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("err:", err)
			return
		}
	} else if fileInfo, err := s.file.Stat(); err == nil && fileInfo.Size() > 100000000 {
		// 文件超过500M，创建新文件
		s.file.Close()
		fileName := createFileName()
		filePath := filepath.Join(path, fileName)
		var err error
		s.file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("err:", err)
			return
		}
	}

	bytes, err := json.Marshal(nodeInfo)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = s.file.Write(bytes)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = s.file.WriteString("\n")
	if err != nil {
		log.Println(err)
	}
}

func createDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
		fmt.Printf("Directory '%s' created.\n", dir)
	} else if err != nil {
		return err
	}
	return nil
}

func (s *Parser) isNodeInfoChanged(old, new *NodeInfo) bool {
	if old.RuntimeStatus != new.RuntimeStatus {
		return true
	}
	if old.HaveAvailableIp != new.HaveAvailableIp {
		return true
	}
	if old.StreamdPorts != new.StreamdPorts {
		return true
	}
	return false
}

func (s *Parser) nodeMonitor() {
	s.allNodeInfoMap = make(map[string]*NodeInfo)
	ticker := time.NewTicker(time.Duration(15) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ipStatusMap := make(map[string]int)
		allNodes, err := dal.GetAllNode(s.redisCli)
		if err != nil {
			log.Fatalln(err)
		}
		for _, node := range allNodes {
			if !node.IsDynamic {
				continue
			}
			nodeInfo := s.buildNodeInfo(node)
			if old, ok := s.allNodeInfoMap[nodeInfo.NodeId]; !ok {
				s.allNodeInfoMap[nodeInfo.NodeId] = nodeInfo
			} else if s.isNodeInfoChanged(old, nodeInfo) {
				if !s.isNodeAvailable(old) && s.isNodeAvailable(nodeInfo) {
					nodeInfo.EndTime = time.Now().Format("2006-01-02 15:04:05")
					s.writeToFile(nodeInfo)
				}
				s.allNodeInfoMap[nodeInfo.NodeId] = nodeInfo
			}
			s.fillIpStatus(ipStatusMap, node)
		}
		s.dynIpMonitor(ipStatusMap)
		deleteOldFiles()
	}
}

func (s *Parser) fillIpStatus(ipStatusMap map[string]int, node *model.RtNode) {
	for _, ipInfo := range node.Ips {
		if !checkIp(ipInfo) {
			continue
		}
		if node.RuntimeStatus != "Serving" {
			ipStatusMap[IpStatusOffline]++
			continue
		}
		if node.IsBanTransProv {
			ipStatusMap[IpStatusBanProv]++
			continue
		}
		if noStreamdPorts(node) {
			ipStatusMap[IpStatusNoPorts]++
			continue
		}
		if ipInfo.Forbidden {
			ipStatusMap[IpStatusForbidden]++
			continue
		}
		if ipInfo.IPStreamProbe.State != model.StreamProbeStateSuccess {
			ipStatusMap[IpStatusStreamProbeStateFail]++
			continue
		}
		if ipInfo.IPStreamProbe.Speed < 12 && ipInfo.IPStreamProbe.MinSpeed < 10 {
			ipStatusMap[IpStatusStreamProbeSpeedFail]++
			continue
		}
		if ipInfo.OutMBps >= ipInfo.MaxOutMBps*0.8 {
			ipStatusMap[IpStatusBwLimit]++
			continue
		}
	}
}

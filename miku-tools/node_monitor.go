package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/qbox/mikud-live/common/model"
	public "github.com/qbox/mikud-live/common/model"
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
	NodeId          string   `json:"nodeid"`
	MachindId       string   `json:"machineid"`
	RuntimeStatus   string   `json:"runtime_status"`
	StreamdPorts    bool     `json:"streamd_ports"`
	HaveAvailableIp bool     `json:"hava_available_ip"`
	ErrIps          []IpInfo `json:"err_ips"`
	TimeStamp       string   `json:"timestamp"`
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
		TimeStamp:     time.Now().Format("2006-01-02 15:04:05"),
		NodeId:        node.Id,
		MachindId:     node.MachineId,
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

func genFileName() string {
	timestamp := time.Now().Format("2006_01_02") // 年月日
	//return fmt.Sprintf("nodeinfo-%s.json", timestamp)
	return timestamp
}

func deleteOldFiles(path string) error {
	cutoff := time.Now().Add(-8 * 24 * time.Hour) // 3天前的日期
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

func (s *Parser) writeToFile(nodeInfo *NodeInfo, path string) {
	createDirIfNotExist(path)
	latestFile, err := findLatestFile(path)
	if err != nil {
		log.Fatalln(err)
		return
	}
	fileName := genFileName()
	if s.file == nil {
		filePath := filepath.Join(path, fileName)
		var err error
		s.file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("err:", err)
			return
		}
	} else if fileName != latestFile {
		s.file.Close()
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
		allNodes, err := public.GetAllRTNodes(s.logger, s.RedisCli)
		if err != nil {
			s.logger.Error().Msgf("[GetAllNode] get all nodes failed, err: %+v, use snapshot", err)
			return
		}
		for _, node := range allNodes {
			if !node.IsDynamic {
				continue
			}
			nodeInfo := s.buildNodeInfo(node)
			if old, ok := s.allNodeInfoMap[nodeInfo.NodeId]; !ok {
				s.allNodeInfoMap[nodeInfo.NodeId] = nodeInfo
			} else if s.isNodeInfoChanged(old, nodeInfo) {
				s.writeToFile(old, path)
				s.allNodeInfoMap[nodeInfo.NodeId] = nodeInfo
			}
			s.fillIpStatus(ipStatusMap, node)
		}
		s.dynIpMonitor(ipStatusMap)
		deleteOldFiles(path)
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

func (s *Parser) writeDataToFile(data, path string) {
	createDirIfNotExist(path)
	latestFile, err := findLatestFile(path)
	if err != nil {
		log.Fatalln(err)
		return
	}
	fileName := genFileName()
	if s.file == nil {
		filePath := filepath.Join(path, fileName)
		var err error
		s.file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("err:", err)
			return
		}
	} else if fileName != latestFile {
		s.file.Close()
		filePath := filepath.Join(path, fileName)
		var err error
		s.file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("err:", err)
			return
		}
	}

	_, err = s.file.Write([]byte(data))
	if err != nil {
		log.Println(err)
		return
	}
	_, err = s.file.WriteString("\n")
	if err != nil {
		log.Println(err)
	}
}

type ForbiddenNode struct {
	Ts       time.Time `json:"ts"`
	OutBw    float64   `json:"outBw"`
	MaxOutBw float64   `json:"maxOutBw"`
	Overflow bool      `json:"overflow"`
}

type Data struct {
	//AbnormalNodes           []map[string]int         `json:"abnormal_nodes"`
	TotalTimeoutNodes int                      `json:"total_timeout_nodes"`
	TopForbiddenNodes map[string]ForbiddenNode `json:"top_forbidden_nodes"`
	//PcdnErrNodes            []map[string]int         `json:"pcdn_err_nodes"`
	TotalErrNodes       int                      `json:"total_err_nodes"`
	PcdnErrFbiddenNodes map[string]ForbiddenNode `json:"pcdn_err_forbidden_nodes"`
	//ConnectFailNodes        []map[string]int         `json:"connect_fail_nodes"`
	TotalConnectFailNodes   int                      `json:"total_connect_fail_nodes"`
	ConnectFailFbiddenNodes map[string]ForbiddenNode `json:"connect_fail_forbidden_nodes"`
}

func (s *Parser) pcdnErrMonitor() {
	ticker := time.NewTicker(time.Duration(s.conf.Interval) * time.Second)
	defer ticker.Stop()

	scheds := []struct {
		Ip     string
		NodeId string
	}{
		{
			Ip:     "10.20.94.40",
			NodeId: "jjh2294",
		},
		{
			Ip:     "10.20.94.41",
			NodeId: "jjh2295",
		},
		{
			Ip:     "10.34.101.29",
			NodeId: "bili-xs9",
		},
		{
			Ip:     "10.34.101.28",
			NodeId: "bili-xs8",
		},
	}

	for range ticker.C {
		for _, sched := range scheds {
			addr := fmt.Sprintf("http://%s:6060/api/v1/dymetrics", sched.Ip)
			resp, err := get(addr)
			if err != nil {
				s.logger.Error().Err(err).Msg("pcdnErrMonitor get err")
				continue
			}
			var respData Data
			if err := json.Unmarshal([]byte(resp), &respData); err != nil {
				log.Println(err)
				continue
			}
			data := struct {
				Ts   time.Time `json:"ts"`
				Data Data      `json:"data"`
			}{
				Ts:   time.Now(),
				Data: respData,
			}
			bytes, err := json.Marshal(&data)
			if err != nil {
				log.Println(err)
				continue
			}
			path := "/tmp/pcdn_err_dump/" + sched.NodeId
			s.writeDataToFile(string(bytes), path)

			deleteOldFiles(path)
		}
	}
}

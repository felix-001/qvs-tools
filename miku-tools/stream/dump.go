package stream

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"middle-source-analysis/config"
	"middle-source-analysis/node"
	"middle-source-analysis/public"
	"os"
	"os/exec"
	"sort"
	"time"

	localUtil "middle-source-analysis/util"

	qconfig "github.com/qiniu/x/config"
)

var hdr = "流ID, 运营商, 大区, 在线人数, 边缘节点个数, ROOT节点个数, 放大比, 边缘节点详情, ROOT节点详情\n"

/*
func (s *Parser) dump() {
	csv := hdr
	cnt := 0
	var totalBw float64
	var totalRelayBw float64

	roomMap := make(map[string][]string)
	roomOnlineMap := make(map[string]int)
	for streamId, streamDetail := range s.streamDetailMap {
		roomId, id := splitString(streamId)
		roomMap[roomId] = append(roomMap[roomId], id)
		cnt++

		for isp, detail := range streamDetail {
			for area, streamInfo := range detail {
				ratio := streamInfo.Bw / streamInfo.RelayBw
				csv += fmt.Sprintf("%s, %s, %s, %d, %d, %d, %.1f, %+v, %+v\n", streamId, isp, area,
					streamInfo.OnlineNum, len(streamInfo.EdgeNodes),
					len(streamInfo.RootNodes), ratio, streamInfo.EdgeNodes,
					streamInfo.RootNodes)
				totalBw += streamInfo.Bw
				totalRelayBw += streamInfo.RelayBw

				roomOnlineMap[roomId] += int(streamInfo.OnlineNum)
			}
		}

	}
	file := fmt.Sprintf("%d.csv", time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}

	/*
		log.Println("cnt:", cnt)
		log.Printf("totalBw: %.1f, totalRelayBw: %.1f, totalRatio: %.1f", totalBw,
			totalRelayBw, totalBw/totalRelayBw)
		log.Println("room count:", len(roomMap))
		for roomId, ids := range roomMap {
			fmt.Println(roomId, ids)
		}
		log.Println("room - onlineNum info")
		for roomId, onlineNum := range roomOnlineMap {
			fmt.Println(roomId, onlineNum)
		}
}
*/

func str2unix(s string) (int64, error) {
	loc, _ := time.LoadLocation("Local")
	the_time, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return the_time.Unix(), nil
}

func str2time(s string) (time.Time, error) {
	loc, _ := time.LoadLocation("Local")
	return time.ParseInLocation("2006-01-02 15:04:05", s, loc)
}

func SaveNodesStatusDetailToCsv(nodeUnavailableDetail map[string][]public.NodeUnavailableDetail, schedInfos []public.SchedInfo, conf *config.Config) {
	csv := "开始时间, 结束时间, 时长, 原因, 详细\n"
	for _, schedInfo := range schedInfos {
		csv += schedInfo.NodeId + "\n"
		details := nodeUnavailableDetail[schedInfo.NodeId]
		for _, detail := range details {
			csv += fmt.Sprintf("%s, %s, %s, %s, %s\n", detail.Start, detail.End, detail.Duration, detail.Reason, detail.Detail)
		}
	}
	file := fmt.Sprintf("%s-nodes-detail-%d.csv", conf.Stream, time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
	cmd := exec.Command("./qup", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

func dumpStream(conf *config.Config) {
	nodeUnavailableDetailMap := GetNodeUnavailableDetail(1)
	streamSchedInfos := GetStreamSchedInfos(conf)
	sort.Slice(streamSchedInfos, func(i, j int) bool {
		return streamSchedInfos[i].StartTime < streamSchedInfos[j].StartTime
	})
	SaveStreamSchedInfosToCsv(streamSchedInfos, conf)
	SaveNodesStatusDetailToCsv(nodeUnavailableDetailMap, streamSchedInfos, conf)

	/*
		nodeDetailMap := make(map[string][]NodeUnavailableDetail)
		start := streamSchedInfos[0].StartTime / 1000
		end := streamSchedInfos[len(streamSchedInfos)-1].StartTime / 1000
		for _, schedInfo := range streamSchedInfos {
			if _, ok := nodeDetailMap[schedInfo.NodeId]; !ok {
				nodeDetailMap[schedInfo.NodeId] = make([]NodeUnavailableDetail, 0)
			}
			details := nodeUnavailableDetailMap[schedInfo.NodeId]
			log.Println("node unavailable details:", len(details))
			for _, detail := range details {
				detailStart, err := str2unix(detail.Start)
				if err != nil {
					log.Println(err)
					continue
				}
				detailEnd, err := str2unix(detail.End)
				if err != nil {
					log.Println(err)
					continue
				}
				if detailEnd > start && detailStart < end {
					detail.Detail = strings.ReplaceAll(detail.Detail, ",", " ")
					nodeDetailMap[schedInfo.NodeId] = append(nodeDetailMap[schedInfo.NodeId], detail)
				}
			}
		}

		s.saveNodeDetailToCsv(nodeDetailMap)
	*/

}

func saveNodeDetailToCsv(nodeDetailMap map[string][]public.NodeUnavailableDetail, conf *config.Config) {
	csv := "节点, 开始时间, 结束时间, reason, detail\n"
	for nodeId, details := range nodeDetailMap {
		for _, detail := range details {
			csv += fmt.Sprintf("%s, %s, %s, %s, %s\n", nodeId, detail.Start,
				detail.End, detail.Reason, detail.Detail)
		}
	}
	file := fmt.Sprintf("%s-node-detail-%d.csv", conf.Stream, time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
	cmd := exec.Command("./qup", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

func SaveStreamSchedInfosToCsv(streamSchedInfos []public.SchedInfo, conf *config.Config) {
	csv := "时间, ConnId, NodeId, MachineId\n"
	for _, schedInfo := range streamSchedInfos {
		timeStr := localUtil.UnixToTimeStr(schedInfo.StartTime / 1000)
		csv += fmt.Sprintf("%s, %s, %s, %s\n", timeStr, schedInfo.ConnId,
			schedInfo.NodeId, schedInfo.MachindId)
	}
	file := fmt.Sprintf("%s-nodes-%d.csv", conf.Stream, time.Now().Unix())
	err := ioutil.WriteFile(file, []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
	cmd := exec.Command("./qup", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("命令执行出错: %v\n", err)
		return
	}
	fmt.Println(string(output))
}

func DumpRoots() {
	areaNodeCntMap := make(map[string]int)
	for key, detail := range node.AllRootNodesMapByAreaIsp {
		cnt := 0
		for _, rootNode := range detail {
			if rootNode.Forbidden {
				continue
			}
			cnt++
		}
		areaNodeCntMap[key] = cnt
	}
	jsonbody, err := json.Marshal(areaNodeCntMap)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(jsonbody))
}

func DumpQPM(conf *config.Config) {
	LastMinuteAreaReqInfo := make(map[string]map[string]time.Time)
	if err := qconfig.LoadFile(&LastMinuteAreaReqInfo, conf.QpmFile); err != nil {
		log.Fatalf("load config failed, err: %v", err)
	}
	for _, isp := range public.Isps {
		for _, area := range public.Areas {
			simpArea, ok := public.AreaMap[area]
			if !ok {
				log.Fatalf("area not found, %s\n", area)
			}
			simpIsp, ok := public.IspMap[isp]
			if !ok {
				log.Fatalf("isp not found, %s", isp)
			}
			key := fmt.Sprintf("%s_%s_%s_%s", conf.Bucket, conf.Stream, simpArea, simpIsp)
			areaUserInfo, ok := LastMinuteAreaReqInfo[key]
			if !ok {
				fmt.Println("key not found:", key)
				continue
			}
			cnt := 0
			for _, lastUpdatedTime := range areaUserInfo {
				if time.Since(lastUpdatedTime) > time.Minute*time.Duration(5) {
					continue
				}
				cnt++
			}
			fmt.Println(key, cnt)
		}
	}
}

func DumpAllNodes() {
	jsonbody, err := json.Marshal(node.AllNodesMap)
	if err != nil {
		log.Println(err)
		return
	}
	os.WriteFile("./allNodes.json", jsonbody, 0644)
	for _, node := range node.AllNodesMap {
		for _, ip := range node.Ips {
			if ip.Forbidden {
				log.Println(node.Id, ip.Ip)
			}
		}
	}
}

package util

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type StreamdQos struct {
	Ts          string
	NodeId      string
	Region      string
	City        string
	StreamName  string
	Type        string
	LagDuration int
	LagCount    int
	LocalAddr   string
	RemoteAddr  string
}

/*
SELECT Ts,NodeID,StreamName, Region,CustomerSource,LocalAddr,RemoteAddr,StartTime,LagCount,LagDuration
from miku_data.streamd_qos
where AppName == 'douyu' and Ts > '2024-09-21 08:19:00' and Ts < '2024-09-21 08:24:00' and LagDuration > 0
*/

// TODO： 通过csv Heaer 确定相应的字段在第几列

func loadLagData() []StreamdQos {
	bytes, err := ioutil.ReadFile(Conf.LagFile)
	if err != nil {
		log.Println("read fail", Conf.LagFile, err)
		return nil
	}
	datas := make([]StreamdQos, 0)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines[1:] {
		fields := strings.Split(line, ",")
		if len(fields) != 10 {
			continue
		}

		lagDuration, err := strconv.ParseInt(fields[8], 10, 32)
		if err != nil {
			log.Println(err)
			continue
		}
		lagCount, err := strconv.ParseInt(fields[9], 10, 32)
		if err != nil {
			log.Println(err)
			continue
		}
		data := StreamdQos{
			Ts:          fields[0],
			NodeId:      fields[1],
			StreamName:  fields[2],
			Region:      fields[3],
			LagDuration: int(lagDuration),
			LagCount:    int(lagCount),
		}
		datas = append(datas, data)
	}
	return datas
}

func LagAnalysis() {
	nodeLagCntMap := make(map[string]int)
	streamLagCntMap := make(map[string]int)
	regionLagCntMap := make(map[string]int)
	areaLagCntMap := make(map[string]int)

	datas := loadLagData()
	for _, data := range datas {
		//nodeLagCntMap[data.NodeId] += data.LagCount
		nodeLagCntMap[data.NodeId] += data.LagDuration
		streamLagCntMap[data.StreamName] += data.LagCount
		regionLagCntMap[data.Region] += data.LagCount
		//area := util.ProvinceAreaRelation(data.Region)
		area := ""
		areaLagCntMap[area] += data.LagCount
	}
	log.Println("node lag map:")
	pairs := SortIntMap(nodeLagCntMap)
	DumpSlice(pairs)
	log.Println("stream lag map:")
	pairs = SortIntMap(streamLagCntMap)
	DumpSlice(pairs)
	log.Println("region lag map:")
	pairs = SortIntMap(regionLagCntMap)
	DumpSlice(pairs)
	log.Println("area lag map:")
	pairs = SortIntMap(areaLagCntMap)
	DumpSlice(pairs)
}

func CoverChk() {
	rows := LocadCsv(Conf.QosFile)
	ipParseErrCnt := 0
	ispNotMatchCnt := 0
	provinceNotMatchCnt := 0
	areaNotMatchCnt := 0
	provinceMap := make(map[string]int)
	areaMap := make(map[string]int)
	csv := "localAddr, 省份, 大区, 运营商, remoteAddr, 省份, 大区, 运营商, result\n"
	for _, row := range rows[1:] {
		localAddr := row[2]
		remoteAddr := row[3]
		parts := strings.Split(localAddr, ":")
		if len(parts) != 2 {
			log.Println("parse local addr err", localAddr)
			continue
		}
		localIp := parts[0]
		parts = strings.Split(remoteAddr, ":")
		if len(parts) != 2 {
			log.Println("parse remote addr err", remoteAddr)
		}
		remoteIp := parts[0]
		localIsp, localArea, localProvince := GetLocate(localIp, IpParser)
		if localIsp == "" || localArea == "" || localProvince == "" {
			ipParseErrCnt++
			continue
		}
		remoteIsp, remoteArea, remoteProvince := GetLocate(remoteIp, IpParser)
		if remoteIsp == "" || remoteArea == "" || remoteProvince == "" {
			ipParseErrCnt++
			continue
		}

		match := true
		result := ""
		if localIsp != remoteIsp {
			match = false
			result += "isp不匹配;"
			ispNotMatchCnt++
		}

		if localProvince != remoteProvince {
			match = false
			result += " 省份不匹配;"
			provinceNotMatchCnt++

			key := remoteProvince + "_" + remoteIsp
			provinceMap[key]++
		}

		if localArea != remoteArea {
			match = false
			result += " 大区不匹配"
			areaNotMatchCnt++

			key := remoteArea + "_" + remoteIsp
			areaMap[key]++

		}

		if !match {
			csv += fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s, %s\n",
				localAddr, localProvince, localArea, localIsp,
				remoteAddr, remoteProvince, remoteArea, remoteIsp,
				result)
		}
	}

	err := ioutil.WriteFile("/tmp/coverResult.csv", []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
	log.Println("ipParseErrCnt:", ipParseErrCnt, "ispNotMatchCnt:", ispNotMatchCnt, "areaNotMatchCnt:", areaNotMatchCnt,
		"provinceNotMatchCnt:", provinceNotMatchCnt)
	pairs := SortIntMap(provinceMap)
	for _, pair := range pairs {
		log.Println(pair.Key, pair.Value)
	}
	pairs = SortIntMap(areaMap)
	for _, pair := range pairs {
		log.Println(pair.Key, pair.Value)
	}
}

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/qbox/mikud-live/cmd/lived/common/util"
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

func (s *Parser) loadLagData() []StreamdQos {
	bytes, err := ioutil.ReadFile(s.conf.LagFile)
	if err != nil {
		log.Println("read fail", s.conf.LagFile, err)
		return nil
	}
	datas := make([]StreamdQos, 0)
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) != 10 {
			continue
		}

		lagDuration, err := strconv.ParseInt(fields[6], 10, 32)
		if err != nil {
			log.Println(err)
			continue
		}
		lagCount, err := strconv.ParseInt(fields[7], 10, 32)
		if err != nil {
			log.Println(err)
			continue
		}
		data := StreamdQos{
			Ts:          fields[0],
			NodeId:      fields[1],
			Region:      fields[2],
			City:        fields[3],
			StreamName:  fields[4],
			Type:        fields[5],
			LagDuration: int(lagDuration),
			LagCount:    int(lagCount),
		}
		datas = append(datas, data)
	}
	return datas
}

func (s *Parser) LagAnalysis() {
	nodeLagCntMap := make(map[string]int)
	streamLagCntMap := make(map[string]int)
	regionLagCntMap := make(map[string]int)
	areaLagCntMap := make(map[string]int)

	datas := s.loadLagData()
	for _, data := range datas {
		nodeLagCntMap[data.NodeId] += data.LagCount
		streamLagCntMap[data.StreamName] += data.LagCount
		regionLagCntMap[data.Region] += data.LagCount
		area := util.ProvinceAreaRelation(data.Region)
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

func (s *Parser) CoverChk() {
	rows := s.locadCsv(s.conf.QosFile)
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
		localIsp, localArea, localProvince := getLocate(localIp, s.ipParser)
		if localIsp == "" || localArea == "" || localProvince == "" {
			s.logger.Error().Str("localIsp", localIsp).
				Str("localArea", localArea).
				Str("localProvince", localProvince).
				Str("localIp", localIp).
				Msg("getLocate")
			ipParseErrCnt++
			continue
		}
		remoteIsp, remoteArea, remoteProvince := getLocate(remoteIp, s.ipParser)
		if remoteIsp == "" || remoteArea == "" || remoteProvince == "" {
			s.logger.Error().Str("remoteIsp", remoteIsp).
				Str("remoteIsp", remoteIsp).
				Str("remoteProvince", remoteProvince).
				Str("remoteIp", remoteIp).
				Msg("getLocate")
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

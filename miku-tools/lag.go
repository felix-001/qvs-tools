package main

import (
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
}

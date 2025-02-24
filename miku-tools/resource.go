package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

// go run . -cmd res -domain mikudns.com -name *.subscribe
func (s *Parser) Res() {
	cli, err := tencent_dnspod.NewTencentClient(s.conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := cli.GetRecords(xl, s.conf.Domain, "", "", 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	lineMap := s.buildLineMap(resp.RecordList)

	areaIspMap := s.areaIspFilter(lineMap)
	fmt.Println("area len:", len(areaIspMap))
	needAreas := s.checkAreaIspCoverage(areaIspMap)
	fmt.Printf("没有覆盖的大区: %+v\n", needAreas)
	//s.dumpAreaIspMapBw(areaIspMap)
	s.dumpAreaIspMapDetail(areaIspMap)

	//areaMap := s.buildAreaMap(areaIspMap)
	//s.dumpAreaMapToCsv(areaMap)

	s.evaluateBw(areaIspMap)
}

func (s *Parser) buildLineMap(recordList []*dnspod.RecordListItem) map[string][]string {
	lineMap := make(map[string][]string)
	for _, record := range recordList {
		if record.Name == nil {
			continue
		}
		if *record.Name != s.conf.Name {
			continue
		}
		if record.Line == nil {
			continue
		}
		if record.Value == nil {
			continue
		}
		if record.Type == nil {
			continue
		}
		if *record.Type == "AAAA" {
			continue
		}
		if *record.Status != "ENABLE" {
			continue
		}
		//fmt.Printf("Name: %s, Line: %s, Value: %s, Type: %s, remark: %s, status: %s\n",
		//*record.Name, *record.Line, *record.Value, *record.Type, *record.Remark, *record.Status)
		lineMap[*record.Line] = append(lineMap[*record.Line], *record.Value)
	}
	return lineMap
}

func (s *Parser) lineRemoveIsp(line string) string {
	for _, isp := range Isps {
		if !strings.Contains(line, isp) {
			continue
		}
		key := strings.ReplaceAll(line, isp, "")
		return key
	}
	return ""
}

func (s *Parser) areaIspFilter(lineMap map[string][]string) map[string][]string {

	out := make(map[string][]string)
	for line, ips := range lineMap {
		area := s.lineRemoveIsp(line)
		if !ContainInStringSlice(area, Areas) {
			continue
		}
		out[line] = ips
	}
	return out
}

func (s *Parser) checkAreaIspCoverage(areaIspMap map[string][]string) []string {
	needAreas := make([]string, 0)
	for _, area := range Areas {
		for _, isp := range Isps {
			areaIsp := area + isp
			if _, ok := areaIspMap[areaIsp]; ok {
				continue
			}
			needAreas = append(needAreas, areaIsp)
		}
	}
	return needAreas
}

func (s *Parser) dumpAreaIspMapDetail(areaMap map[string][]string) {
	ipNodeMap := s.buildIpNodeMap()
	idcs := make(map[string]bool)
	for line, ips := range areaMap {
		fmt.Printf("%s\n", line)
		//fmt.Printf("\t%+v\n", ips)
		for _, ip := range ips {
			node := ipNodeMap[ip]
			if node == nil {
				fmt.Println("ip:", ip, "node not found")
				continue
			}
			fmt.Printf("\t%s %s, %s\n", ip, node.Id, node.Idc)
			idcs[node.Idc] = true
		}
	}
	fmt.Printf("idcs: ")
	for idc := range idcs {
		fmt.Printf("%s ", idc)
	}
	println()
}

func (s *Parser) dumpAreaIspMapBw(areaMap map[string][]string) {
	total := 0
	for line, ips := range areaMap {
		fmt.Printf("%s\n", line)
		fmt.Printf("\t节点个数: %d, 带宽: %dG\n", len(ips), len(ips)*3)
		total += len(ips) * 3
	}
	fmt.Printf("总建设带宽: %dG\n", total)
}

func (s *Parser) getAreaIdcsMap(areaMap map[string][]string) map[string]map[string]bool {
	ipNodeMap := s.buildIpNodeMap()
	areaIdcsMap := make(map[string]map[string]bool) // key1: area key2: idc
	for area, ips := range areaMap {
		for _, ip := range ips {
			node := ipNodeMap[ip]
			if node == nil {
				fmt.Println("ip:", ip, "node not found")
				continue
			}
			if _, ok := areaIdcsMap[area]; !ok {
				areaIdcsMap[area] = make(map[string]bool)
			}
			areaIdcsMap[area][node.Idc] = true
		}
	}
	return areaIdcsMap
}

func (s *Parser) getAreaNodesMap(areaMap map[string][]string) map[string]map[string]bool {
	ipNodeMap := s.buildIpNodeMap()
	areaNodessMap := make(map[string]map[string]bool) // key1: area key2: nodeId
	for area, ips := range areaMap {
		for _, ip := range ips {
			node := ipNodeMap[ip]
			if node == nil {
				fmt.Println("ip:", ip, "node not found")
				continue
			}
			if _, ok := areaNodessMap[area]; !ok {
				areaNodessMap[area] = make(map[string]bool)
			}
			areaNodessMap[area][node.Id] = true
		}
	}
	return areaNodessMap
}

func (s *Parser) dumpAreaMapToCsv(areaMap map[string][]string) {
	areaIdcsMap := s.getAreaIdcsMap(areaMap)
	areaNodesMap := s.getAreaNodesMap(areaMap)
	csv := fmt.Sprintf("大区运营商, 节点个数, 节点列表, idcs, 建设带宽, 需要带宽, 比例(总共:%dG), 缺失带宽\n", s.conf.TotoalNeedBw)
	total := 0
	nodeCnt := 0
	defaultRatio := float64(1) / float64(len(Areas))
	fmt.Printf("平均每个大区的占比: %.2f\n", defaultRatio)
	for area, ips := range areaMap {
		idcsStr := ""
		for idc := range areaIdcsMap[area] {
			idcsStr += idc + "   "
		}
		nodesStr := ""
		for nodeId := range areaNodesMap[area] {
			nodesStr += nodeId + "   "
		}
		ratio, ok := s.conf.BwRatioConfig[area]["total"]
		if !ok {
			ratio = defaultRatio
		}

		maxBw := len(areaNodesMap[area]) * 10
		needBw := int(float64(s.conf.TotoalNeedBw) * ratio)
		missBw := needBw - maxBw
		if missBw < 0 {
			missBw = 0
		}
		csv += fmt.Sprintf("%s, %d, %s, %s, %dG, %dG, %.2f, %dG\n", area, len(areaNodesMap[area]), nodesStr, idcsStr,
			maxBw, needBw, ratio, missBw)
		total += len(ips) * 10
		nodeCnt += len(ips)
	}
	fmt.Printf("总建设带宽: %dG, 总节点个数: %d\n", total, nodeCnt)
	err := os.WriteFile("/tmp/static_idc_bw.csv", []byte(csv), 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// key: area value: ips
func (s *Parser) buildAreaMap(areaIspMap map[string][]string) map[string][]string {
	areaMap := make(map[string][]string)
	for areaIsp, ips := range areaIspMap {
		area := s.lineRemoveIsp(areaIsp)
		areaMap[area] = append(areaMap[area], ips...)
	}
	return areaMap
}

func (s *Parser) buildIpNodeMap() map[string]*model.RtNode {
	ipNodeMap := make(map[string]*model.RtNode)
	for _, node := range s.allNodesMap {
		for _, ip := range node.Ips {
			ipNodeMap[ip.Ip] = node
		}
	}
	fmt.Println("ipNodeMap len:", len(ipNodeMap))
	return ipNodeMap
}

func (s *Parser) evaluateBw(areaIspMap map[string][]string) {
	areaIspIdcsMap := s.getAreaIdcsMap(areaIspMap)
	csv := fmt.Sprintf("大区运营商, 节点个数, idcs, 比例(总共: %dG), 需要带宽, 节点建设带宽, idc建设带宽, 缺失带宽, 备注\n", s.conf.TotoalNeedBw)
	defaultAreaRatio := float64(1) / float64(len(Areas))
	defaultIspRatio := float64(1) / float64(len(Isps))
	fmt.Printf("平均每个大区的占比: %.2f, 平均每个isp占比: %.2f\n", defaultAreaRatio, defaultIspRatio)
	for areaIsp, ips := range areaIspMap {
		idcs, ok := areaIspIdcsMap[areaIsp]
		if !ok {
			s.logger.Error().Str("areaIsp", areaIsp).Msg("evaluateBw, get idcs err")
			continue
		}
		area := s.lineRemoveIsp(areaIsp)
		areaRatio, ok := s.conf.BwRatioConfig[area]["total"]
		if !ok {
			s.logger.Error().Str("areaIsp", areaIsp).Msg("area ratio not configed")
			areaRatio = defaultAreaRatio
		}
		isp := areaIsp[len(area):]
		ispRatio, ok := s.conf.BwRatioConfig[area][isp]
		if !ok {
			s.logger.Error().Str("area", area).Str("isp", isp).Msg("isp ratio not configed")
			ispRatio = defaultIspRatio
		}
		ratio := areaRatio * ispRatio
		needBw := int(float64(s.conf.TotoalNeedBw) * ratio)

		idcMaxBw := 0
		idcsStr := ""
		for idc := range idcs {
			idcMaxBw += s.conf.IdcBwConfig[idc][isp]
			idcsStr += idc + "   "
		}
		nodeMaxBw := len(ips) * 3
		missBw := needBw - nodeMaxBw
		if missBw < 0 {
			missBw = 0
		}

		csv += fmt.Sprintf("%s, %d, %s, %.2f, %d, %d, %d, %d\n",
			areaIsp, len(ips), idcsStr, ratio, needBw, nodeMaxBw, idcMaxBw, missBw)
	}
	err := os.WriteFile("/tmp/static_idc_bw.csv", []byte(csv), 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
}

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

	areaMap := s.buildAreaMap(areaIspMap)
	s.dumpAreaMapToCsv(areaMap)
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

func (s *Parser) dumpAreaMapToCsv(areaIspMap map[string][]string) {
	defaultRatio := float64(1) / float64(len(Areas))
	fmt.Printf("平均每个大区的占比: %.2f\n", defaultRatio)
	total := 0
	nodeCnt := 0
	csv := "大区运营商, 节点个数, 建设带宽\n"
	for areaIsp, ips := range areaIspMap {
		//ratio := s.conf.BwRatioConfig[areaIsp]
		csv += fmt.Sprintf("%s, %d, %dG\n", areaIsp, len(ips), len(ips)*3)
		total += len(ips) * 3
		nodeCnt += len(ips)
	}
	fmt.Printf("总建设带宽: %dG, 总节点个数: %d\n", total, nodeCnt)
	err := os.WriteFile("/tmp/static_idc_bw.csv", []byte(csv), 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
}

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

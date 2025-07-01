package node

import (
	"context"
	"fmt"
	"middle-source-analysis/public"
	localUtil "middle-source-analysis/util"
	"os"
	"strings"

	"github.com/qbox/mikud-live/cmd/dnspod/tencent_dnspod"
	"github.com/qbox/mikud-live/common/model"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
	"github.com/rs/zerolog"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

var (
	logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
)

// go run . -cmd res -domain mikudncom -name *.subscribe
func Res() {
	cli, err := tencent_dnspod.NewTencentClient(Conf.DnsPod)
	if err != nil {
		fmt.Println(err)
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := cli.GetRecords(xl, Conf.Domain, "", "", 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	lineMap := buildLineMap(resp.RecordList)

	areaIspMap := areaIspFilter(lineMap)
	fmt.Println("area len:", len(areaIspMap))
	needAreas := checkAreaIspCoverage(areaIspMap)
	fmt.Printf("没有覆盖的大区: %+v\n", needAreas)
	//dumpAreaIspMapBw(areaIspMap)
	dumpAreaIspMapDetail(areaIspMap)

	//areaMap := buildAreaMap(areaIspMap)
	//dumpAreaMapToCsv(areaMap)

	evaluateBw(areaIspMap)
}

func buildLineMap(recordList []*dnspod.RecordListItem) map[string][]string {
	lineMap := make(map[string][]string)
	for _, record := range recordList {
		if record.Name == nil {
			continue
		}
		if *record.Name != Conf.Name {
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

func lineRemoveIsp(line string) string {
	for _, isp := range public.Isps {
		if !strings.Contains(line, isp) {
			continue
		}
		key := strings.ReplaceAll(line, isp, "")
		return key
	}
	return ""
}

func areaIspFilter(lineMap map[string][]string) map[string][]string {

	out := make(map[string][]string)
	for line, ips := range lineMap {
		area := lineRemoveIsp(line)
		if !localUtil.ContainInStringSlice(area, public.Areas) {
			continue
		}
		out[line] = ips
	}
	return out
}

func checkAreaIspCoverage(areaIspMap map[string][]string) []string {
	needAreas := make([]string, 0)
	for _, area := range public.Areas {
		for _, isp := range public.Isps {
			areaIsp := area + isp
			if _, ok := areaIspMap[areaIsp]; ok {
				continue
			}
			needAreas = append(needAreas, areaIsp)
		}
	}
	return needAreas
}

func dumpAreaIspMapDetail(areaMap map[string][]string) {
	ipNodeMap := buildIpNodeMap()
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

func DumpAreaIspMapBw(areaMap map[string][]string) {
	total := 0
	for line, ips := range areaMap {
		fmt.Printf("%s\n", line)
		fmt.Printf("\t节点个数: %d, 带宽: %dG\n", len(ips), len(ips)*3)
		total += len(ips) * 3
	}
	fmt.Printf("总建设带宽: %dG\n", total)
}

func getAreaIdcsMap(areaMap map[string][]string) map[string]map[string]bool {
	ipNodeMap := buildIpNodeMap()
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

func getAreaNodesMap(areaMap map[string][]string) map[string]map[string]bool {
	ipNodeMap := buildIpNodeMap()
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

func dumpAreaMapToCsv(areaMap map[string][]string) {
	areaIdcsMap := getAreaIdcsMap(areaMap)
	areaNodesMap := getAreaNodesMap(areaMap)
	csv := fmt.Sprintf("大区运营商, 节点个数, 节点列表, idcs, 建设带宽, 需要带宽, 比例(总共:%dG), 缺失带宽\n", Conf.TotoalNeedBw)
	total := 0
	nodeCnt := 0
	defaultRatio := float64(1) / float64(len(public.Areas))
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
		ratio, ok := Conf.BwRatioConfig[area]["total"]
		if !ok {
			ratio = defaultRatio
		}

		maxBw := len(areaNodesMap[area]) * 10
		needBw := int(float64(Conf.TotoalNeedBw) * ratio)
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
func BuildAreaMap(areaIspMap map[string][]string) map[string][]string {
	areaMap := make(map[string][]string)
	for areaIsp, ips := range areaIspMap {
		area := lineRemoveIsp(areaIsp)
		areaMap[area] = append(areaMap[area], ips...)
	}
	return areaMap
}

func buildIpNodeMap() map[string]*model.RtNode {
	ipNodeMap := make(map[string]*model.RtNode)
	for _, node := range AllNodesMap {
		for _, ip := range node.Ips {
			ipNodeMap[ip.Ip] = node
		}
	}
	fmt.Println("ipNodeMap len:", len(ipNodeMap))
	return ipNodeMap
}

func evaluateBw(areaIspMap map[string][]string) {
	areaIspIdcsMap := getAreaIdcsMap(areaIspMap)
	csv := fmt.Sprintf("大区运营商, 节点个数, idcs, 比例(总共: %dG), 需要带宽, 节点建设带宽, idc建设带宽, 缺失带宽, 备注\n", Conf.TotoalNeedBw)
	defaultAreaRatio := float64(1) / float64(len(public.Areas))
	defaultIspRatio := float64(1) / float64(len(public.Isps))
	fmt.Printf("平均每个大区的占比: %.2f, 平均每个isp占比: %.2f\n", defaultAreaRatio, defaultIspRatio)
	for areaIsp, ips := range areaIspMap {
		idcs, ok := areaIspIdcsMap[areaIsp]
		if !ok {
			logger.Error().Str("areaIsp", areaIsp).Msg("evaluateBw, get idcs err")
			continue
		}
		area := lineRemoveIsp(areaIsp)
		areaRatio, ok := Conf.BwRatioConfig[area]["total"]
		if !ok {
			logger.Error().Str("areaIsp", areaIsp).Msg("area ratio not configed")
			areaRatio = defaultAreaRatio
		}
		isp := areaIsp[len(area):]
		ispRatio, ok := Conf.BwRatioConfig[area][isp]
		if !ok {
			logger.Error().Str("area", area).Str("isp", isp).Msg("isp ratio not configed")
			ispRatio = defaultIspRatio
		}
		ratio := areaRatio * ispRatio
		needBw := int(float64(Conf.TotoalNeedBw) * ratio)

		idcMaxBw := 0
		idcsStr := ""
		for idc := range idcs {
			idcMaxBw += Conf.IdcBwConfig[idc][isp]
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

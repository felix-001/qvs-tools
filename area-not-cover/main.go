package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
)

type Pair struct {
	Key   string
	Value int
}

func not_cover() {
	b, err := ioutil.ReadFile("/Users/liyuanquan/Downloads/grafana_data_export.csv")
	if err != nil {
		log.Println("read fail", "/Users/liyuanquan/Downloads/grafana_data_export.csv", err)
		return
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	i := 0
	//data := []Data{}
	data := map[string]int{}
	for scanner.Scan() {
		line := scanner.Text()
		if i == 0 {
			i++
			continue
		}
		ss := strings.Split(line, ";")
		val, err := strconv.Atoi(ss[2])
		if err != nil {
			log.Fatal(err)
		}
		if val > 0 {
			//log.Println(ss[1], ss[0], ss[2])
			data[ss[0]] += val

		}
		i++
	}

	var pairs []Pair
	for key, value := range data {
		pairs = append(pairs, Pair{key, value})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	for _, pair := range pairs {
		fmt.Printf("%s: %d\n", pair.Key, pair.Value)
	}
}

func ProvinceAreaRelation(province string) string {
	switch province {
	case "黑龙江", "吉林", "辽宁":
		return "东北"
	case "北京", "天津", "河北", "山西", "内蒙古":
		return "华北"
	case "河南", "湖北", "湖南":
		return "华中"
	case "山东", "江苏", "安徽", "上海", "浙江", "江西", "福建":
		return "华东"
	case "广东", "广西", "海南":
		return "华南"
	case "陕西", "甘肃", "宁夏", "青海", "新疆":
		return "西北"
	case "四川", "贵州", "云南", "重庆", "西藏":
		return "西南"
	case "香港", "澳门", "台湾":
		return "其它"
	default:
		return ""
	}
}

func AreaProvinceRelation(area string) []string {
	switch area {
	case "东北":
		return []string{"黑龙江", "吉林", "辽宁"}
	case "华北":
		return []string{"北京", "天津", "河北", "山西", "内蒙"}
	case "华中":
		return []string{"河南", "湖北", "湖南"}
	case "华东":
		return []string{"山东", "江苏", "安徽", "上海", "浙江", "江西", "福建"}
	case "华南":
		return []string{"广东", "广西", "海南"}
	case "西北":
		return []string{"陕西", "甘肃", "宁夏", "青海", "新疆"}
	case "西南":
		return []string{"四川", "贵州", "云南", "重庆", "西藏"}
	case "其它":
		return []string{"香港", "澳门", "台湾"}
	default:
		return nil
	}
}

type Info struct {
	Percent int
	Users   int
}

func douyu_user() {
	//b, err := ioutil.ReadFile("/Users/liyuanquan/Downloads/grafana_data_export-douyu-user-yd.csv")
	//b, err := ioutil.ReadFile("/Users/liyuanquan/Downloads/grafana_data_export_lt.csv")
	b, err := ioutil.ReadFile("/Users/liyuanquan/Downloads/grafana_data_export_dx.csv")
	if err != nil {
		log.Println("read fail", "grafana_data_export-douyu-user-yd.csv", err)
		return
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	i := 0
	//data := []Data{}
	data := map[string]int{}
	for scanner.Scan() {
		line := scanner.Text()
		if i == 0 {
			i++
			continue
		}
		i++
		ss := strings.Split(line, ";")
		province := strings.Trim(ss[0], `"`)
		user, err := strconv.Atoi(ss[2])
		if err != nil {
			log.Fatal(err)
		}
		data[province] += user
	}

	var pairs []Pair
	for key, value := range data {
		pairs = append(pairs, Pair{key, value})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	region := make(map[string]map[string]*Info)
	for _, pair := range pairs {
		fmt.Printf("%s: %d\n", pair.Key, pair.Value)
		area := ProvinceAreaRelation(pair.Key)
		if region[area] == nil { // 检查内层map是否已初始化
			region[area] = make(map[string]*Info) // 如果没有，初始化内层map
		}
		//info := Info{Users: pair.Value}
		if region[area][pair.Key] == nil {
			region[area][pair.Key] = &Info{}
		}
		region[area][pair.Key].Users += pair.Value
	}
	//log.Printf("%+v\n", region)
	for area, provinces := range region {
		log.Printf("%s ", area)
		total := 0
		for _, info := range provinces {
			//log.Printf("%s %d", province, info)
			total += info.Users
		}
		for _, info := range provinces {
			info.Percent = info.Users * 100 / total
			//region[area][province].Percent =
		}
		for province, info := range provinces {
			fmt.Printf("%s 用户数: %d 百分比: %d%%\n", province, info.Users, info.Percent)
			total += info.Users
		}
		fmt.Println()
	}

}

// 各个大区当前可用带宽
/*
func bps() {
	b, err := ioutil.ReadFile("bps.csv")
	if err != nil {
		log.Println("read fail", "bps.csv", err)
		return
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	for scanner.Scan() {
		line := scanner.Text()
		ss := strings.Split(line, ",")
		sss := strings.Split(ss[0], "_")
		region := sss[0]
		isp := sss[1]
	}
}
*/

// 运营商各个大区用户占比
func userDistribution(f, isp string, percent float64) string {
	b, err := ioutil.ReadFile(f)
	if err != nil {
		log.Println("read fail", "bps.csv", err)
		return ""
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	data := map[string]int{}
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if i == 0 {
			i++
			continue
		}
		ss := strings.Split(line, ";")
		if _, ok := data[ss[0]]; !ok {
			users, err := strconv.Atoi(ss[2])
			if err != nil {
				log.Fatal(err)
			}
			province := strings.Trim(ss[0], `"`)
			data[province] = users
		}
		i++
	}

	log.Println(data)
	rigionMap := map[string]int{}
	for province, usercount := range data {
		region := ProvinceAreaRelation(province)
		if region != "" {
			rigionMap[region] += usercount
		} else {
			log.Println("region is empty, province:", province)
		}
	}
	log.Println(rigionMap)

	var regionSlice []Pair
	for k, v := range rigionMap {
		regionSlice = append(regionSlice, Pair{Key: k, Value: v})
	}
	sort.Slice(regionSlice, func(i, j int) bool {
		return regionSlice[i].Value > regionSlice[j].Value
	})
	log.Println(regionSlice)

	/*
		csv := ""
		for _, s := range regionSlice {
			region := s.Key
			total := s.Value
			provinces := AreaProvinceRelation(region)
			for _, province := range provinces {
				csv += fmt.Sprintf("%s, %s, %d, %d%%\n", region, province, data[province], data[province]*100/total)
			}
			csv += fmt.Sprintf("%s, total, %d, 100%%\n", region, total)
		}
	*/
	allTotal := 0
	csv := ""
	for _, s := range regionSlice {
		allTotal += s.Value
	}
	for _, s := range regionSlice {
		csv += fmt.Sprintf("%s, %s, %d, %d%%, %.2fG\n", isp, s.Key, s.Value, s.Value*100/allTotal, 150*(float64(s.Value)/float64(allTotal))*percent)
	}
	return csv
}

func allDistribution() {
	csv := "运营商, 大区, 用户数, 占比, 需要带宽\n"

	txt := userDistribution("dx.csv", "电信", 0.46)
	csv += txt

	txt = userDistribution("用户分布-移动.csv", "移动", 0.35)
	csv += txt

	txt = userDistribution("用户分布-联通.csv", "联通", 0.18)
	csv += txt

	err := ioutil.WriteFile("user-distribution.csv", []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
}

// 各个大区可用带宽
func useableBandwidth() {
	b, err := ioutil.ReadFile("bps5.csv")
	if err != nil {
		log.Println("read fail", "bps5.csv", err)
		return
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	csv := ""
	for scanner.Scan() {
		line := scanner.Text()
		ss := strings.Split(line, ",")
		bwStr := strings.Trim(ss[1], " ")
		bw, err := strconv.ParseFloat(bwStr, 64)
		if err != nil {
			fmt.Println("Error converting string to float64:", err)
			return
		}
		bwG := bw * 8 / 1024
		sss := strings.Split(ss[0], "_")
		csv += fmt.Sprintf("%s, %s, %.2fG\n", sss[1], sss[0], bwG)
	}
	err = ioutil.WriteFile("bps-result.csv", []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
}

func nodeDistribution() {
	b, err := ioutil.ReadFile("nodeinfo6.csv")
	if err != nil {
		log.Println("read fail", "nodeinfo6.csv", err)
		return
	}
	lines := strings.Split(string(b), "\n")
	// 获取所有isp+area的总节点数
	ispAreaTotalNodes := map[string]int{}
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "Total") {
			ss := strings.Split(lines[i], ",")
			key := ss[1] + "_" + ss[0]
			nodeCntStr := strings.Trim(ss[3], " ")
			nodeCount, err := strconv.ParseInt(nodeCntStr, 10, 32)
			if err != nil {
				log.Println("Error converting string to int:", err)
				return
			}
			ispAreaTotalNodes[key] = int(nodeCount)
		}
	}
	csv := ""
	for i := 0; i < len(lines)-1; i++ {
		ss := strings.Split(lines[i], ",")
		if len(ss) < 3 {
			log.Fatalln(lines[i], i)
		}
		nodeCntStr := strings.Trim(ss[3], " ")
		nodeCount, err := strconv.ParseInt(nodeCntStr, 10, 32)
		if err != nil {
			log.Println("Error converting string to int:", err)
			return
		}
		key := ss[1] + "_" + ss[0]
		total := ispAreaTotalNodes[key]
		percent := float64(nodeCount*100) / float64(total)
		csv += fmt.Sprintf("%s, %s, %s, %s, %.2f%%\n", ss[0], ss[1], ss[2], ss[3], percent)

	}
	err = ioutil.WriteFile("node-distribution.csv", []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}

}

type Province struct {
	UserCount        int
	UserPercent      float64
	UserPercentInIsp float64
	NodeCount        int
	NodePercent      float64
	NodePercentInIsp float64
	FreeBw           float64
	NeedBw           float64
}

type Area struct {
	Name      string
	Provinces []Province
}

var isps = []string{"移动", "联通", "电信"}

func loadfile(isp string) string {
	file := ""
	switch isp {
	case "移动":
		file = "yd.csv"
	case "联通":
		file = "lt.csv"
	case "电信":
		file = "dx.csv"
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println("read fail", file, err)
		return ""
	}
	return string(b)
}

func readfile(f string) string {
	b, err := ioutil.ReadFile(f)
	if err != nil {
		log.Println("read fail", f, err)
		return ""
	}
	return string(b)
}

type Data struct {
	Key   string
	Value Province
}

func sortMapData(raw map[string]Province) []Data {
	var datas []Data
	for key, value := range raw {
		datas = append(datas, Data{key, value})
	}

	sort.Slice(datas, func(i, j int) bool {
		return datas[i].Value.UserCount > datas[j].Value.UserCount
	})
	return datas
}

func dumpISP(isp string, areas map[string]map[string]Province) {
	csv := "大区, 省份, 用户数, 用户数在大区占比, 用户数在isp占比, 节点数, 节点数在大区占比, 节点数在isp占比\n"
	for area, provinceInfo := range areas {
		for province, info := range provinceInfo {
			//log.Println(isp, area, province, info.UserCount, info.UserPercent, info.NodeCount, info.NodePercent)
			if province != "合计" {
				csv += fmt.Sprintf("%s, %s, %d, %.1f%%, %.1f%%, %d, %.1f%%, %.1f%%\n",
					area, province, info.UserCount, info.UserPercent, info.UserPercentInIsp,
					info.NodeCount, info.NodePercent, info.NodePercentInIsp)
			}
		}
		info := provinceInfo["合计"]
		csv += fmt.Sprintf("%s, %s, %d, %.1f%%, %.1f%%, %d, %.1f%%, %.1f%%\n",
			area, "合计", info.UserCount, info.UserPercent, info.UserPercentInIsp,
			info.NodeCount, info.NodePercent, info.NodePercentInIsp)
	}
	err := ioutil.WriteFile(isp+".csv", []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}
}

func calcUserPercentInArea(areas map[string]map[string]Province) {
	for _, provinceInfo := range areas {
		for province, info := range provinceInfo {
			totalInfo := provinceInfo["合计"]
			info.UserPercent = float64(info.UserCount*100) / float64(totalInfo.UserCount)
			provinceInfo[province] = info
		}
	}
}

func parseIspUserDistribution(isp string) (int, map[string]map[string]Province) {
	totalUserCount := 0
	areas := map[string]map[string]Province{}
	raw := loadfile(isp)
	if raw == "" {
		log.Fatalln("load file err")
	}
	lines := strings.Split(raw, "\r\n")
	for _, line := range lines[1:] {
		ss := strings.Split(line, ";")
		if len(ss) != 3 {
			log.Fatalln("item not 3")
		}
		province := strings.ReplaceAll(ss[0], `"`, "")
		area := ProvinceAreaRelation(province)
		if province == "空" {
			continue
		}
		userCount, err := strconv.ParseInt(ss[2], 10, 32)
		if err != nil {
			log.Fatalln(err)
		}
		if provinces, ok := areas[area]; !ok {
			provinces := map[string]Province{
				province: Province{UserCount: int(userCount)},
				"合计":     Province{UserCount: int(userCount)},
			}
			areas[area] = provinces

			totalUserCount += int(userCount)
		} else if _, ok := provinces[province]; !ok {
			provinces[province] = Province{UserCount: int(userCount)}
			info := provinces["合计"]
			info.UserCount += int(userCount)
			provinces["合计"] = info

			totalUserCount += int(userCount)
		}
	}
	return totalUserCount, areas
}

// 移动_东北_黑龙江 --> 3
func parseIspNodeDistribution() (int, map[string]int) {
	nodeInfo := map[string]int{}
	totalNodeCount := 0
	raw := readfile("nodeinfo.csv")
	if raw == "" {
		log.Fatalln("read file err")
	}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		ss := strings.Split(line, ",")
		if len(ss) != 4 {
			log.Fatalln("parse line err", line)
		}
		isp := strings.Trim(ss[0], " ")
		area := strings.Trim(ss[1], " ")
		province := strings.Trim(ss[2], " ")
		nodeCountStr := strings.Trim(ss[3], " ")
		key := isp + "_" + area + "_" + province
		nodeCount, err := strconv.ParseInt(nodeCountStr, 10, 32)
		if err != nil {
			log.Fatalln(err)
		}
		nodeInfo[key] = int(nodeCount)
		if province != "Total" {
			totalNodeCount += int(nodeCount)
		}
	}
	return totalNodeCount, nodeInfo
}

func mergeData(isp string, totalUserCount int, areas map[string]map[string]Province, totalNodeCount int, nodeInfo map[string]int) {
	for area, provinces := range areas {
		totalKey := isp + "_" + area + "_Total"
		total := nodeInfo[totalKey]
		for province, info := range provinces {
			key := isp + "_" + area + "_" + province
			if province == "合计" {
				key = totalKey
			}
			info.NodeCount = nodeInfo[key]
			info.NodePercent = float64(info.NodeCount*100) / float64(total)
			info.NodePercentInIsp = float64(info.NodeCount*100) / float64(totalNodeCount)
			info.UserPercentInIsp = float64(info.UserCount*100) / float64(totalUserCount)
			provinces[province] = info
		}
	}
}

// 运营商 --> 大区 --> 省份
func douyuData() {
	totalNodeCount, nodeInfo := parseIspNodeDistribution()
	for _, isp := range isps {
		totalUsercount, areas := parseIspUserDistribution(isp)
		calcUserPercentInArea(areas)
		//log.Println(areas)
		mergeData(isp, totalUsercount, areas, totalNodeCount, nodeInfo)
		dumpISP(isp, areas)
	}

}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//douyu_user()
	//bps()
	//allDistribution()
	//useableBandwidth()
	//nodeDistribution()
	douyuData()
	//log.Println(nodeInfo)
}

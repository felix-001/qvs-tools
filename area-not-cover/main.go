package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qbox/pili/common/ipdb.v1"
	qconfig "github.com/qiniu/x/config"
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
	UserCount          int
	UserPercent        float64
	UserPercentInIsp   float64
	NodeCount          int
	NodePercent        float64
	NodePercentInIsp   float64
	NodeIpCount        int
	NodeIpPercent      float64
	NodeIpPercentInIsp float64
	FreeBw             float64
	NeedBw             float64
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
	log.Println(file)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println("read fail", file, err)
		return ""
	}
	res := strings.ReplaceAll(string(b), ";", ",")
	res = strings.ReplaceAll(res, "\"", "")
	return res
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

type AreaData struct {
	Key   string
	Value map[string]Province
}

func sortAreaMapData(raw map[string]map[string]Province) []AreaData {
	var datas []AreaData
	for key, value := range raw {
		datas = append(datas, AreaData{key, value})
	}
	sort.Slice(datas, func(i, j int) bool {
		return datas[i].Value["合计"].UserCount > datas[j].Value["合计"].UserCount
	})
	return datas
}

func dumpISP(isp string, areas map[string]map[string]Province) string {
	//csv := "运营商, 大区, 省份, 用户数, 用户数在大区占比, 用户数在isp占比, 节点数, 节点数在大区占比, 节点数在isp占比, ip数, ip数在大区占比, ip数在isp占比, 需要带宽(Gbps), 可用带宽(Gbps), 是否需要增加带宽\n"
	areaDatas := sortAreaMapData(areas)
	var bwNeed float64 = 0
	switch isp {
	case "电信":
		bwNeed = 200 * 0.4
	case "移动":
		bwNeed = 200 * 0.3
	case "联通":
		bwNeed = 200 * 0.3
	}
	csv := ""
	for _, areaData := range areaDatas {
		log.Println(areaData.Key)
		area := areaData.Key
		provinceInfo := areaData.Value
		//provinces := sortMapData(provinceInfo)
		//for _, data := range provinces {
		//province := data.Key
		//info := data.Value
		/*
			if province != "合计" {
				csv += fmt.Sprintf("%s, %s, %s, %d, %.1f%%, %.1f%%, %d, %.1f%%, %.1f%%, %d, %.1f%%, %.1f%%, , %.1f\n",
					isp, area, province, info.UserCount, info.UserPercent, info.UserPercentInIsp,
					info.NodeCount, info.NodePercent, info.NodePercentInIsp, info.NodeIpCount,
					info.NodeIpPercent, info.NodeIpPercentInIsp, info.FreeBw)
			}
		*/
		//}
		info := provinceInfo["合计"]
		ispAreaBwNeed := bwNeed * info.NodeIpPercentInIsp / 100
		needAddBw := "否"
		if ispAreaBwNeed > info.FreeBw {
			needAddBw = "是"
		}
		csv += fmt.Sprintf("%s, %s, %d, %.1f%%, %.1f%%, %d, %.1f%%, %.1f%%, %d, %.1f%%, %.1f%%, %.1f,%.1f, %s\n",
			isp, area, info.UserCount, info.UserPercent, info.UserPercentInIsp,
			info.NodeCount, info.NodePercent, info.NodePercentInIsp, info.NodeIpCount,
			info.NodeIpPercent, info.NodeIpPercentInIsp, ispAreaBwNeed, info.FreeBw, needAddBw)
	}
	return csv
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
	log.Println("load isp", isp)
	raw := loadfile(isp)
	if raw == "" {
		log.Fatalln("load file err")
	}
	lines := strings.Split(raw, "\r\n")
	log.Println("lines", len(lines))
	for _, line := range lines[1:] {
		ss := strings.Split(line, ",")
		if len(ss) != 3 {
			log.Fatalln("item not 3", line, len(ss))
		}
		province := strings.ReplaceAll(ss[0], `"`, "")
		area := ProvinceAreaRelation(province)
		if province == "空" || area == "" {
			continue
		}
		userCount, err := strconv.ParseInt(ss[2], 10, 32)
		if err != nil {
			log.Fatalln(err)
		}
		if provinces, ok := areas[area]; !ok {
			log.Println(area)
			provinces := map[string]Province{
				province: {UserCount: int(userCount)},
				"合计":     {UserCount: int(userCount)},
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
func parseIspNodeDistribution() (map[string]int, map[string]int) {
	nodeInfo := map[string]int{}
	totalNodeCount := map[string]int{}
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
		if province != "Total" && province != "" {
			//log.Println(province, nodeCount)
			//totalNodeCount += int(nodeCount)
			totalNodeCount[isp] += int(nodeCount)
		}
	}
	return totalNodeCount, nodeInfo
}

func mergeData(isp string, totalUserCount int, areas map[string]map[string]Province, nodesData map[string]int, bwData map[string]float64) {
	totalIspIp := nodesData[isp+"_ip"]
	for area, provinces := range areas {
		totalKey := isp + "_" + area
		total := nodesData[totalKey]
		totalIp := nodesData[totalKey+"_ip"]
		for province, info := range provinces {
			key := isp + "_" + area + "_" + province
			if province == "合计" {
				key = totalKey
			}
			info.NodeCount = nodesData[key]
			if total != 0 {
				info.NodePercent = float64(info.NodeCount*100) / float64(total)
			}

			info.NodePercentInIsp = float64(info.NodeCount*100) / float64(nodesData[isp])
			info.UserPercentInIsp = float64(info.UserCount*100) / float64(totalUserCount)

			ipKey := key + "_ip"
			info.NodeIpCount = nodesData[ipKey]
			if totalIp != 0 {
				info.NodeIpPercent = float64(info.NodeIpCount*100) / float64(totalIp)
			}
			info.NodeIpPercentInIsp = float64(info.NodeIpCount*100) / float64(totalIspIp)

			if province == "合计" {
				key = isp + "_" + area
			}
			info.FreeBw = bwData[key]

			provinces[province] = info
		}
	}
}

// 运营商 --> 大区 --> 省份
func douyuData(ipParser *ipdb.City) {
	//totalNodeCount, nodeInfo := parseIspNodeDistribution()
	dnsRecord := getDnsRecord()
	nodesData, bwData := getNodesData(ipParser, dnsRecord)
	log.Println("总节点数:", totalNodeCount)
	csv := "运营商, 大区, 用户数, 用户数在大区占比, 用户数在isp占比, 节点数, 节点数在大区占比, 节点数在isp占比, ip数, ip数在大区占比, ip数在isp占比, 需要带宽(Gbps), 可用带宽(Gbps), 是否需要增加带宽\n"
	for _, isp := range isps {
		totalUsercount, areas := parseIspUserDistribution(isp)
		fmt.Printf("%+v\n", areas)
		log.Println("len:", len(areas))
		calcUserPercentInArea(areas)
		//log.Println(areas)
		mergeData(isp, totalUsercount, areas, nodesData, bwData)
		csv += dumpISP(isp, areas)
	}
	err := ioutil.WriteFile("斗鱼带宽需求.csv", []byte(csv), 0644)
	if err != nil {
		log.Println(err)
	}

}

func RunCmd(cmdstr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	fmt.Println(cmd)
	//cmd.Stderr = os.Stderr
	b, err := cmd.CombinedOutput()
	if err != nil {
		return string(b), err
	}
	//return string(b), nil
	raw := string(b)
	//log.Println(raw)
	if strings.Contains(raw, "Pseudo-terminal") {
		new := ""
		ss := strings.Split(raw, "\n")
		if len(ss) == 1 {
			return "", nil
		}
		for _, str := range ss {
			if strings.Contains(str, "Pseudo-terminal") {
				continue
			}
			if len(str) == 0 {
				continue
			}
			//log.Println("str len:", len(str))
			new += str + "\r\n"
		}
		//log.Println("new:", new)
		return new, nil
	}
	return raw, nil
}

func jumpboxCmd(rawCmd string) (string, error) {
	jumpbox := "ssh -t liyuanquan@10.20.34.27"
	cmd := fmt.Sprintf("%s \" %s \"", jumpbox, rawCmd)
	return RunCmd(cmd)
}

type NodeStatus string

const (
	NODE_PENDING    NodeStatus = "pending"
	NODE_NORMAL     NodeStatus = "normal"
	NODE_DISABLED   NodeStatus = "disabled"
	NODE_UNKNOWN    NodeStatus = "unknown"
	NODE_ANY_STATUS NodeStatus = ""
)

type Ability struct {
	Can    bool `json:"can" bson:"can"`
	Frozen bool `json:"frozen" bson:"frozen"`
}

type IpIsp struct {
	Ip         string `bson:"ip" json:"ip"`
	Isp        string `bson:"isp" json:"isp"`
	Forbidden  bool   `bson:"forbidden,omitempty" json:"forbidden,omitempty"` // 单个 ip 封禁
	ExternalIP string `bson:"externalIP,omitempty" json:"externalIP,omitempty"`

	IsIPv6 bool `bson:"is_ipv6" json:"is_ipv6"`
}

type Node struct {
	Id           string             `bson:"_id" json:"id"`
	Idc          string             `bson:"idc" json:"idc"`
	Provider     string             `bson:"provider" json:"provider"`
	HostName     string             `bson:"host" json:"host"`
	BandwidthMbs float64            `bson:"bwMbps" json:"bwMbps"`
	LanIP        string             `bson:"lanIP" json:"lanIP"`
	Status       NodeStatus         `bson:"status" json:"status"`
	Abilities    map[string]Ability `bson:"abilities" json:"abilities,omitempty"` // fixme: key 形式待定
	IpIsps       []IpIsp            `bson:"ipisps" json:"ipisps"`
	Comment      string             `bson:"comment" json:"comment"`
	UpdateTime   int64              `bson:"updateTime" json:"updateTime"`
	IsDynamic    bool               `bson:"isDynamic" json:"isDynamic"`
	IsMixture    bool               `bson:"isMixture" json:"isMixture"`
	MachineId    string             `bson:"machineId" json:"machineId"`
	RecordTime   time.Time          `bson:"recordTime" json:"recordTime"`
	// fixme: oauth 相关信息记录
}

type IPStreamProbe struct {
	State      int     `json:"state"`      // 定义同 `StreamProbeState`
	Speed      float64 `json:"speed"`      // Mbps
	UpdateTime int64   `json:"updateTime"` // s

	// for sliding window
	SlidingSpeeds [10]float64 `json:"slidingSpeeds"`
	MinSpeed      float64     `json:"minSpeed"`
}

type RtIpStatus struct {
	IpIsp
	Interface  string  `json:"interface"`  // 网卡名称
	InMBps     float64 `json:"inMBps"`     // 入流量带宽，单位：MBps
	OutMBps    float64 `json:"outMBps"`    // 出流量带宽，单位：MBps
	MaxInMBps  float64 `json:"maxInMBps"`  // 下行建设带宽, 单位：MBps
	MaxOutMBps float64 `json:"maxOutMBps"` // 上行建设带宽，单位：MBps

	IPStreamProbe IPStreamProbe `json:"ipStreamProbe"`
}

type StreamdPorts struct {
	Http    int `json:"http" bson:"http"`       // http, ws
	Https   int `json:"https" bson:"https"`     // https, wss
	Wt      int `json:"wt" bson:"wt"`           // wt, quic
	Rtmp    int `json:"rtmp" bson:"rtmp"`       // rtmp
	Control int `json:"control" bson:"control"` // control
}

type ServiceStatus struct {
	CPU    float64 `json:"cpu"` // cpu usage percent
	RSS    int64   `json:"rss"` // resident set memory size
	FD     int     `json:"fd"`
	MaxFD  int     `json:"max_fd"`
	Uptime int64   `json:"uptime"`
}

type RtNode struct {
	Node

	// runtime info
	RuntimeStatus string                   `json:"runtimeStatus"`
	Ips           []RtIpStatus             `json:"ips"`
	StreamdPorts  StreamdPorts             `json:"streamdPorts"`
	Services      map[string]ServiceStatus `json:"services"`
}

var (
	notServingNodeCount  = 0
	portErrNodeCount     = 0
	forbiddenCount       = 0
	privateIpCount       = 0
	streamProbeFailCount = 0
	speedErrCount        = 0
	freeBwErrIpCount     = 0
	totalNodeCount       = 0
	totalIpCount         = 0
	noDnsRecordCount     = 0
	ipv6Count            = 0
	allIpCount           = 0
	canFrozenCount       = 0
	servicesCount        = 0
)

func check(ip RtIpStatus, dnsRecord map[string]DynamicIpRecord) bool {
	if ip.Forbidden {
		forbiddenCount++
		return false
	}

	if net.ParseIP(ip.Ip).IsPrivate() {
		privateIpCount++
		return false
	}
	if ip.IPStreamProbe.State != 1 {
		streamProbeFailCount++
		return false
	}

	if ip.IPStreamProbe.Speed < 20 &&
		ip.IPStreamProbe.MinSpeed < 10 {
		speedErrCount++
		return false
	}

	if ip.OutMBps >= ip.MaxOutMBps*0.9 {
		freeBwErrIpCount++
		return false
	}

	if _, ok := dnsRecord[ip.Ip]; !ok {
		noDnsRecordCount++
		return false
	}
	if ip.IsIPv6 {
		ipv6Count++
		return false
	}
	return true
}

func checkNode(node RtNode) bool {
	if node.RuntimeStatus != "Serving" {
		notServingNodeCount++
		return false
	}
	if node.StreamdPorts.Http <= 0 || node.StreamdPorts.Https <= 0 || node.StreamdPorts.Wt <= 0 {
		portErrNodeCount++
		return false
	}
	ability, ok := node.Abilities["live"]
	if !ok || !ability.Can || ability.Frozen {
		canFrozenCount++
		return false
	}

	if _, ok = node.Services["live"]; !ok {
		servicesCount++
		return false
	}
	return true
}

func dump() {
	log.Println("not servint node count:", notServingNodeCount)
	log.Println("port err node count:", portErrNodeCount)
	log.Println("forbidden count:", forbiddenCount)
	log.Println("private ip count:", privateIpCount)
	log.Println("stream probe fail count:", streamProbeFailCount)
	log.Println("speed err count:", speedErrCount)
	log.Println("total node count:", totalNodeCount)
	log.Println("total ip count:", totalIpCount)
	log.Println("free bw not enough ip count:", freeBwErrIpCount)
	log.Println("no dns record count:", noDnsRecordCount)
	log.Println("ipv6 count:", ipv6Count)
	log.Println("all ip count:", allIpCount)
	log.Println("can frozen count:", canFrozenCount)
	log.Println("services count:", servicesCount)
}

func getProvinceName(ip string, ipParser *ipdb.City) string {
	locate, err := ipParser.Find(ip)
	if err != nil {
		log.Fatalln(err)
	}
	return locate.Region
}

func dumpData(data map[string]int) {
	for key, value := range data {
		log.Println(key, value)
	}
}

func getBw(isp, area, provinceName string, ip RtIpStatus, bwData map[string]float64) {
	provinceKey := fmt.Sprintf("%s_%s_%s", isp, area, provinceName)
	freeBw := (ip.MaxOutMBps*0.85 - ip.OutMBps) * 8 / 1024
	bwData[provinceKey] += freeBw
	areaKey := fmt.Sprintf("%s_%s", isp, area)
	bwData[areaKey] += freeBw
}

func getNodeIpData(isp, area, provinceName string, nodeData map[string]int) {
	provinceIpKey := fmt.Sprintf("%s_%s_%s_ip", isp, area, provinceName)
	nodeData[provinceIpKey] += 1
	areaIpKey := fmt.Sprintf("%s_%s_ip", isp, area)
	nodeData[areaIpKey] += 1
	ispKey := fmt.Sprintf("%s_ip", isp)
	nodeData[ispKey] += 1
	totalIpCount++
}

func getNodeData(isp, area, provinceName string, nodeData map[string]int) {
	provinceKey := fmt.Sprintf("%s_%s_%s", isp, area, provinceName)
	nodeData[provinceKey] += 1
	areaKey := fmt.Sprintf("%s_%s", isp, area)
	nodeData[areaKey] += 1
	nodeData[isp] += 1
	totalNodeCount++
}

func calcData(isp string, nodes []RtNode, ipParser *ipdb.City, bwData map[string]float64, nodeData map[string]int, dnsRecord map[string]DynamicIpRecord) {
	for _, node := range nodes {
		if !checkNode(node) {
			continue
		}
		provinceName := ""
		area := ""
		for _, ip := range node.Ips {
			allIpCount++
			if !check(ip, dnsRecord) {
				continue
			}
			if ip.Isp != isp {
				continue
			}
			provinceName = getProvinceName(ip.Ip, ipParser)
			area = ProvinceAreaRelation(provinceName)
			getBw(isp, area, provinceName, ip, bwData)
			getNodeIpData(isp, area, provinceName, nodeData)
		}
		if provinceName != "" {
			getNodeData(isp, area, provinceName, nodeData)
		}
	}
}

func getNodesData(ipParser *ipdb.City, dnsRecord map[string]DynamicIpRecord) (map[string]int, map[string]float64) {
	s, err := jumpboxCmd("curl -s http://10.34.139.33:2240/v1/runtime/nodes?dynamic=true")
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("nodes.json", []byte(s), 0644)
	if err != nil {
		log.Println(err)
	}
	nodes := []RtNode{}
	if err := json.Unmarshal([]byte(s), &nodes); err != nil {
		log.Println(err)
		return nil, nil
	}
	log.Println("total nodes:", len(nodes))
	bwData := map[string]float64{}
	nodeData := map[string]int{}
	for _, isp := range isps {
		calcData(isp, nodes, ipParser, bwData, nodeData, dnsRecord)
	}
	//dumpData(bwData)
	//dumpData(nodeData)
	dump()
	return nodeData, bwData
}

type DynamicIpRecord struct {
	Ip         string    `json:"ip"`
	RecordType string    `json:"type"`
	Value      string    `json:"value"`
	Domain     string    `json:"domain"`
	CommitId   string    `json:"commitId"`
	CreateTime time.Time `json:"createTime"`
}

func getDnsRecord() map[string]DynamicIpRecord {
	cmd := "echo \"hgetall miku_ip_domain_dns_map\" | redis-cli-5 -x -h 10.20.54.24 -p 8200 -c --raw"
	resp, err := jumpboxCmd(cmd)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println(resp)
	dnsRecord := map[string]DynamicIpRecord{}
	lines := strings.Split(resp, "\r\n")

	for i := 1; i < len(lines)-1; i += 2 {
		record := DynamicIpRecord{}
		if err := json.Unmarshal([]byte(lines[i+1]), &record); err != nil {
			log.Fatalln(err, lines[i+1])
		}
		dnsRecord[lines[i]] = record
	}
	//log.Println(dnsRecord)
	/*
		for k, v := range dnsRecord {
			log.Println("key is :", k, "val is:", v)
		}
	*/
	log.Println("dns record len:", len(dnsRecord))
	return dnsRecord
}

var (
	configFile = flag.String("f", "/usr/local/etc/miku.json", "the config file")
)

type Config struct {
	IPDB ipdb.Config `json:"ipdb"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conf := &Config{}
	err := qconfig.LoadFile(conf, *configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s\n", err.Error())
	}
	ipParser, err := ipdb.NewCity(conf.IPDB)
	if err != nil {
		log.Fatalf("[IPDB NewCity] err: %+v\n", err)
	}
	douyuData(ipParser)
	//douyu_user()
	//bps()
	//allDistribution()
	//useableBandwidth()g	//nodeDistribution()
	//log.Println(nodeInfo)
}

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
	case "北京", "天津", "河北", "山西", "内蒙":
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

func main() {
	douyu_user()
}

package main

import (
	"fmt"
	"os"
)

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
func main() {
	data := AreaProvinceRelation(os.Args[1])
	if len(data) != 0 {
		fmt.Println(data)
		return
	}
	area := ProvinceAreaRelation(os.Args[1])
	if len(os.Args) == 3 {
		area2 := ProvinceAreaRelation(os.Args[2])
		fmt.Println(area, area2)

	} else {
		fmt.Println(area)
	}
}

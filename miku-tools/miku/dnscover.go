package miku

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"mikutool/public/util"

	schedUtil "github.com/qbox/mikud-live/cmd/sched/common/util"
)

// dnsCoverCSVHeader 探测 CSV 的表头，用于跳过表头行
const dnsCoverCSVHeader = "探测点"

// dnsCoverProvinces 省份前缀列表，按长度倒序匹配，避免「内蒙」抢走「内蒙古」
var dnsCoverProvinces = []string{
	"黑龙江", "内蒙古", "内蒙", "北京", "天津", "河北", "山西", "吉林", "辽宁",
	"河南", "湖北", "湖南", "山东", "江苏", "安徽", "上海", "浙江", "江西",
	"广东", "广西", "海南", "福建", "陕西", "甘肃", "宁夏", "青海", "新疆",
	"四川", "贵州", "云南", "重庆", "西藏", "香港", "澳门", "台湾",
}

// dnsCoverIsps ISP 后缀列表，按长度倒序匹配
var dnsCoverIsps = []string{"鹏博士", "教育网", "长城宽带", "铁通", "移动", "联通", "电信", "广电"}

type dnsCoverStat struct {
	probe      string
	probeIP    string
	dns        string
	probeArea  string
	probeIsp   string
	resolveIP  string
	resolveProv string
	resolveIsp string
	resolveArea string
	areaIsp    string
	matched    bool
	valid      bool
	reason     string
}

// DnsCover 读取探测结果 CSV，逐行比对「探测点所属大区+ISP」与「解析结果第一个 IP 所属大区+ISP」，
// 输出不一致的探测点，并统计大区ISP覆盖率。
// CSV 列: 探测点,探测源IP,DNS服务器,解析时间,解析结果,预期解析结果
func (m *Miku) DnsCover() {
	if m.conf.F == "" {
		log.Println("缺少探测结果文件，请使用 -f <csv_file>")
		return
	}
	if m.resources.IpParser == nil {
		log.Println("ipdb 未加载，无法解析 IP 归属")
		return
	}

	f, err := os.Open(m.conf.F)
	if err != nil {
		log.Printf("打开文件失败: %v\n", err)
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	var stats []dnsCoverStat
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("读取 CSV 行失败: %v\n", err)
			break
		}
		if len(record) == 0 {
			continue
		}
		probe := strings.TrimSpace(strings.TrimPrefix(record[0], "\ufeff"))
		if probe == "" || probe == dnsCoverCSVHeader {
			continue
		}

		stat := dnsCoverStat{probe: probe}
		if len(record) > 1 {
			stat.probeIP = strings.TrimSpace(record[1])
		}
		if len(record) > 2 {
			stat.dns = strings.TrimSpace(record[2])
		}
		var results string
		if len(record) > 4 {
			results = record[4]
		}

		province := probeProvince(probe)
		isp := probeIsp(probe)
		if province == "" || isp == "" {
			stat.reason = "探测点无法识别省份/ISP"
			stats = append(stats, stat)
			continue
		}
		area, _ := schedUtil.ProvinceAreaRelation(province)
		if area == "" {
			stat.reason = fmt.Sprintf("省份 %s 无大区映射", province)
			stats = append(stats, stat)
			continue
		}
		stat.probeArea = area
		stat.probeIsp = isp
		stat.areaIsp = area + isp

		ip := firstIP(results)
		if ip == "" {
			stat.reason = "解析结果中未找到 IP"
			stats = append(stats, stat)
			continue
		}
		stat.resolveIP = ip

		_, rIsp, rArea, rProv := util.GetLocate(ip, m.resources.IpParser)
		if rProv == "" && rIsp == "" {
			stat.reason = "IP 库无法解析该 IP"
			stats = append(stats, stat)
			continue
		}
		rIsp = normalizeIsp(rIsp)
		stat.resolveProv = rProv
		stat.resolveIsp = rIsp
		stat.resolveArea = rArea
		stat.valid = true
		stat.matched = rArea == area && rIsp == isp
		stats = append(stats, stat)
	}

	printDnsCover(stats)
}

// probeProvince 从探测点名称中提取省份，如「贵州贵阳移动」-> 贵州
func probeProvince(probe string) string {
	for _, p := range dnsCoverProvinces {
		if strings.HasPrefix(probe, p) {
			return p
		}
	}
	return ""
}

// probeIsp 从探测点名称中提取 ISP，如「贵州贵阳移动」-> 移动
func probeIsp(probe string) string {
	for _, isp := range dnsCoverIsps {
		if strings.HasSuffix(probe, isp) {
			return isp
		}
	}
	return ""
}

// normalizeIsp 归一化 ISP 名称，兼容「中国移动」「移动通信」等写法
func normalizeIsp(isp string) string {
	isp = strings.TrimSpace(isp)
	isp = strings.TrimPrefix(isp, "中国")
	for _, name := range dnsCoverIsps {
		if strings.HasPrefix(isp, name) {
			return name
		}
	}
	return isp
}

// firstIP 从解析结果（多行文本）中取第一个合法 IP
func firstIP(results string) string {
	lines := strings.FieldsFunc(results, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if ip := net.ParseIP(line); ip != nil {
			return line
		}
	}
	return ""
}

func printDnsCover(stats []dnsCoverStat) {
	var mismatches []dnsCoverStat
	var valid, matched int
	areaTotal := make(map[string]int)
	areaMatch := make(map[string]int)

	for _, s := range stats {
		if !s.valid {
			continue
		}
		valid++
		areaTotal[s.areaIsp]++
		if s.matched {
			matched++
			areaMatch[s.areaIsp]++
		} else {
			mismatches = append(mismatches, s)
		}
	}

	fmt.Printf("\n总行数: %d, 有效行: %d, 一致: %d, 不一致: %d\n", len(stats), valid, matched, len(mismatches))

	if len(mismatches) > 0 {
		fmt.Println("\n=== 大区ISP 与解析结果不一致的探测点 ===")
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "探测点\t探测大区ISP\t解析IP\t解析省份\t解析ISP\t解析大区ISP\t探测源IP")
		for _, s := range mismatches {
			resolveAreaIsp := s.resolveArea + s.resolveIsp
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				s.probe, s.areaIsp, s.resolveIP, s.resolveProv, s.resolveIsp, resolveAreaIsp, s.probeIP)
		}
		tw.Flush()
	}

	var invalid []dnsCoverStat
	for _, s := range stats {
		if !s.valid {
			invalid = append(invalid, s)
		}
	}
	if len(invalid) > 0 {
		fmt.Println("\n=== 无法比对的行 ===")
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "探测点\t原因")
		for _, s := range invalid {
			fmt.Fprintf(tw, "%s\t%s\n", s.probe, s.reason)
		}
		tw.Flush()
	}

	fmt.Println("\n=== 各大区ISP 覆盖率 ===")
	areas := make([]string, 0, len(areaTotal))
	for a := range areaTotal {
		areas = append(areas, a)
	}
	sort.Slice(areas, func(i, j int) bool {
		if areaTotal[areas[i]] != areaTotal[areas[j]] {
			return areaTotal[areas[i]] > areaTotal[areas[j]]
		}
		return areas[i] < areas[j]
	})
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, a := range areas {
		fmt.Fprintf(tw, "%s\t%.2f%%\t(%d/%d)\n",
			a, float64(areaMatch[a])*100/float64(areaTotal[a]), areaMatch[a], areaTotal[a])
	}
	tw.Flush()

	if valid > 0 {
		fmt.Printf("\n大区ISP覆盖率: %.2f%% (%d/%d)\n", float64(matched)*100/float64(valid), matched, valid)
	} else {
		fmt.Println("\n大区ISP覆盖率: 无有效数据")
	}
}

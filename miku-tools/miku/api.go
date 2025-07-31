package miku

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"mikutool/public/util"
	"mikutool/resources"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	publicUtil "github.com/qbox/mikud-live/common/util"
)

type PlaycheckReq struct {
	Bucket   string            `json:"bucket"`
	Key      string            `json:"key"`
	Url      string            `json:"url"`
	Remote   string            `json:"remoteAddr"`
	Local    string            `json:"localAddr"`
	Node     string            `json:"nodeId"`
	ConnId   string            `json:"connectId"`
	Master   string            `json:"master"`
	Protocol string            `json:"protocol"` // 协议类型，非必填，目前仅hls协议传递
	Headers  map[string]string `json:"headers"`
	User     string            `json:"user"`
}

type PlayCheckResp struct {
	Uid        uint32 `json:"uid"`
	ErrCode    string `json:"code"`
	Message    string `json:"message"`
	ConnectId  string `json:"connectId"`
	Url302     string `json:"redirectUrl"`
	RewriteUrl string `json:"rewriteUrl"`
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	RegTsM3u8  string `json:"regTsM3u8"`  // m3u8文件中ts格式，替换其中的${app}/${stream}/${file}
	FlowMethod int    `json:"flowMethod"` // 计量方式: 1: miku计量系统; 2: pili计量系统; 其它值miku&pili计量系统
}

func playcheck(ip string, conf *config.Config) *PlayCheckResp {
	scheme := "http"
	if conf.Https {
		scheme += "s"
	}
	playUrl := fmt.Sprintf("%s://%s/%s/%s.%s?did=a75e6982-7538-4629-ad3c-fd0d60b1ba54&expire=0",
		scheme, conf.Domain, conf.App, conf.Stream, conf.Format)
	req := PlaycheckReq{
		Bucket:   conf.Bucket,
		Key:      conf.Stream,
		Url:      playUrl,
		Node:     conf.Node,
		Remote:   ip,
		ConnId:   conf.ConnId,
		User:     conf.User,
		Protocol: conf.Protocol,
	}
	fmt.Printf("req: %+v\n", req)
	bytes, err := json.Marshal(&req)
	if err != nil {
		log.Println(err)
		return nil
	}
	var resp PlayCheckResp
	addr := fmt.Sprintf("http://%s:6060/api/v1/playcheck", conf.SchedIp)
	if err := util.Post(addr, string(bytes), &resp); err != nil {
		log.Println(err)
		return nil
	}
	return &resp
}

func Playcheck(conf *config.Config) {
	remote := conf.Ip + ":8080"
	if publicUtil.IsIPv6(conf.Ip) {
		remote = fmt.Sprintf("[%s]:8080", conf.Ip)
	}
	resp := playcheck(remote, conf)
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

type Miku struct {
	conf      *config.Config
	resources resources.Resources
}

func NewMiku() *Miku {
	return &Miku{}
}

func (m *Miku) SetConf(conf *config.Config) {
	m.conf = conf
}

func (m *Miku) SetResources(resources resources.Resources) {
	m.resources = resources
}

func (m *Miku) DumpIps() {
	for isp, ips := range m.resources.V6Ips {
		fmt.Println(isp)
		for prov, ip := range ips {
			fmt.Printf("\tprov: %s, ip: %s\n", prov, ip)
		}
	}
	for isp, ips := range m.resources.V4Ips {
		fmt.Println(isp)
		for prov, ip := range ips {
			fmt.Printf("\tprov: %s, ip: %s\n", prov, ip)
		}
	}
}

func (m *Miku) LoopPlaycheck() {
	for _, ips := range m.resources.V6Ips {
		for _, ip := range ips {
			remote := ip + ":8080"
			if publicUtil.IsIPv6(ip) {
				remote = fmt.Sprintf("[%s]:8080", ip)
			}
			resp := playcheck(remote, m.conf)
			bytes, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				log.Println(err)
				return
			}
			fmt.Println(string(bytes))
			if publicUtil.IsIPv6(ip) {
				u, err := url.Parse(resp.Url302)
				if err != nil {
					log.Println(err)
					continue
				}
				// 处理 IPv6 地址，提取 IP 和端口
				if strings.HasPrefix(u.Host, "[") && strings.Contains(u.Host, "]") {
					idx := strings.Index(u.Host, "]")
					ip := u.Host[1:idx]
					if !publicUtil.IsIPv6(ip) {
						log.Println("invalid ip, not ipv6")
						continue
					}
				} else {
					log.Println("invalid ipv6", u.Host)
					continue
				}
			}
		}
	}

}

// TingYunErrNodes 发送 HTTP 请求获取听云网络数据
func (m *Miku) TingYunErrNodes() {
	if m.conf.Key == "" {
		log.Println("tingyun key is empty")
		return
	}
	// 若 StartTime 或 EndTime 为空，设置默认值
	if m.conf.StartTime == "" || m.conf.EndTime == "" {
		currentTime := time.Now()
		m.conf.EndTime = currentTime.Format("2006-01-02 15:04")
		m.conf.StartTime = currentTime.AddDate(0, 0, -1).Format("2006-01-02 15:04")
	}
	query := url.Values{}
	query.Add("authkey", m.conf.Key)
	query.Add("taskId", m.conf.Task)
	query.Add("beginTimeStr", m.conf.StartTime)
	query.Add("endTimeStr", m.conf.EndTime)
	query.Add("taskType", "3")
	addr := fmt.Sprintf("https://network.tingyun.com/network-report-data/rawdata/rawdata-csv-authkey?%s", query.Encode())
	log.Println("addr:", addr)
	resp, err := http.Get(addr)
	if err != nil {
		log.Printf("请求听云数据失败: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应数据失败: %v", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("请求失败，状态码: %d，响应内容: %s", resp.StatusCode, string(body))
		return
	}

	//fmt.Println(string(body))
	err = os.WriteFile("/tmp/tingyun.csv", body, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	m.statisticsErrNodes(string(body))
}

type ProbeResult struct {
	ClientIp string
	Locate   string
	WaitTime float64
	Time     string
}

type ClientipResult struct {
	PcdnIp   string
	Locate   string
	WaitTime float64
	Time     string
}

func (m *Miku) statisticsErrNodes(data string) {
	lines := strings.Split(data, "\n")
	log.Println("lines:", len(lines))
	ipResultsMap := make(map[string][]ProbeResult)
	clientIpResultMap := make(map[string][]ClientipResult)
	citypIspMap := make(map[string]int)
	pcdnIpAbnormalResultsMap := make(map[string][]ProbeResult)
	for _, line := range lines[1:] {
		fields := strings.Split(line, ",")
		if len(fields) < 27 {
			log.Println("fields:", len(fields))
			continue
		}
		// 去除 fields 中每个元素前后的双引号
		for i := range fields {
			fields[i] = strings.Trim(fields[i], "\"")
		}
		// 错误代码
		if fields[26] != "" {
			log.Println("err code:", fields[26])
			continue
		}
		waitTime, err := strconv.ParseFloat(fields[10], 64)
		if err != nil {
			log.Printf("将 fields[10] 转换为 float64 失败: %v", err)
			continue
		}
		pcdnIp := fields[7]
		ipResultsMap[pcdnIp] = append(ipResultsMap[pcdnIp], ProbeResult{
			ClientIp: fields[3],
			Locate:   fields[2],
			WaitTime: waitTime,
			Time:     fields[0],
		})
		clientIpResultMap[fields[3]] = append(clientIpResultMap[fields[3]], ClientipResult{
			PcdnIp:   pcdnIp,
			Locate:   fields[2],
			WaitTime: waitTime,
			Time:     fields[0],
		})
		citypIspMap[fields[2]]++
		if waitTime > 5 {
			pcdnIpAbnormalResultsMap[pcdnIp] = append(pcdnIpAbnormalResultsMap[pcdnIp], ProbeResult{
				ClientIp: fields[3],
				Locate:   fields[2],
				WaitTime: waitTime,
				Time:     fields[0],
			})
		}
	}
	log.Println("ipResultsMap:", len(ipResultsMap))
	for ip, results := range ipResultsMap {
		fmt.Printf("ip: %s, cnt: %d\n", ip, len(results))
	}
	ipAvgWaitTimeMap := make(map[string]float64)
	single := 0
	multi := 0
	for ip, results := range ipResultsMap {
		var totalWaitTime float64
		for _, result := range results {
			totalWaitTime += result.WaitTime
		}
		ipAvgWaitTimeMap[ip] = totalWaitTime / float64(len(results))
		if ipAvgWaitTimeMap[ip] > 10 {
			fmt.Printf("ip: %s, avg wait time: %.1f\n", ip, ipAvgWaitTimeMap[ip])
			for _, result := range results {
				fmt.Printf("\ttime: %s, client ip: %s, locate: %s, wait time: %.1f\n", result.Time, result.ClientIp, result.Locate, result.WaitTime)
			}
		}
		if len(results) >= 2 {
			multi++
		} else {
			single++
		}
	}
	log.Println("clientIpResultMap:", len(clientIpResultMap))
	for ip, results := range clientIpResultMap {
		fmt.Printf("client ip: %s, cnt: %d\n", ip, len(results))
		for _, result := range results {
			fmt.Printf("\ttime: %s, pcdn ip: %s, locate: %s, wait time: %.1f\n", result.Time, result.PcdnIp, result.Locate, result.WaitTime)
		}
	}
	log.Println("single:", single)
	log.Println("multi:", multi)
	log.Println("citypIspMap:", len(citypIspMap))
	for city, cnt := range citypIspMap {
		fmt.Printf("city: %s, cnt: %d\n", city, cnt)
	}
	log.Println("pcdnIpAbnormalResultsMap:", len(pcdnIpAbnormalResultsMap))
	for ip, results := range pcdnIpAbnormalResultsMap {
		if len(results) < 2 {
			continue
		}
		fmt.Printf("ip: %s, cnt: %d\n", ip, len(results))
		for _, result := range results {
			fmt.Printf("\ttime: %s, client ip: %s, locate: %s, wait time: %.1f\n", result.Time, result.ClientIp, result.Locate, result.WaitTime)
		}
	}
	tasks, err := m.getTaskList()
	if err != nil {
		log.Println("err:", err)
		return
	}
	for _, task := range tasks {
		fmt.Printf("task: %s, url: %s, id: %d, name: %s, expire: %s\n", task.Name, task.Url, task.ID, task.Name, task.Expire)
	}
}

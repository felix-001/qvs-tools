package users

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func getMd5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func HyAuth(conf *config.Config) {
	// 将Unix时间戳转换为十六进制小写字符串
	t := time.Now().Unix() + 3600 // 300天
	timestamp := strconv.FormatInt(t, 16)
	// 拼接固定部分和变动部分
	dataToHash := "huya.com@live/huyalive/" + conf.Stream + ".flv" + timestamp
	log.Println("dataToHash:", dataToHash)
	//wsSecret := fmt.Sprintf("%x", dataToHash)

	wsSecret := getMd5Hash(dataToHash)

	// 生成请求地址
	u := "http://test-qn.flv.huya.com/huyalive/" + conf.Stream + ".flv?wsSecret=" + wsSecret + "&wsTime=" + timestamp
	log.Println(u)

	// 对url进行urlEncode编码
	encodedUrl := url.QueryEscape(u)

	sourceUrl := fmt.Sprintf("http://test-backsrc.huya.com/cdngw/backstreamurl/%s.flv?wsSecret=%s&wsTime=%s"+
		"&isp=cnc&prov=guangdong&city=guangzhou&clientip=45.23.33.2&requesturl=%s",
		conf.Stream, wsSecret, timestamp, encodedUrl)
	log.Println("sourceUrl:", sourceUrl)

	hyP2pAuth(conf)
}

func hyP2pAuth(conf *config.Config) {
	// 假设配置对象和流名称，实际使用时需要替换为真实值
	// 将Unix时间戳转换为十六进制小写字符串
	s := "2025-08-11 11:43:01"
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		log.Printf("解析时间失败: %v", err)
		return
	}
	wsTime := strconv.FormatInt(t.Unix(), 16)
	// 拼接生成wsSecret所需的数据
	dataToHash := "huya.com@live/huyalive/" + conf.Stream + ".pull" + wsTime
	log.Println("dataToHash:", dataToHash)
	wsSecret := getMd5Hash(dataToHash)

	// 生成拉流URL
	pullUrl := fmt.Sprintf("http://pullstream.huya.com/%s.pull?wsSecret=%s&wsTime=%s&from=ali",
		conf.Stream, wsSecret, wsTime)
	log.Println("CDN拉流URL (" + "ali" + "): " + pullUrl)

	// 发起 HTTP GET 请求获取拉流内容
	resp, err := http.Get(pullUrl)
	if err != nil {
		log.Printf("请求拉流 URL 失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("请求拉流 URL 返回状态码异常: %d", resp.StatusCode)
		return
	}

	// 这里可以添加处理响应体的逻辑，例如读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应体失败: %v", err)
		return
	}
	log.Printf("拉流响应内容: %s", string(body))
}

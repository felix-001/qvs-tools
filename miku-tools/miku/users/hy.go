package users

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mikutool/config"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	u := "http://qn.flv.huya.com/src/" + conf.Stream + ".flv?wsSecret=" + wsSecret + "&wsTime=" + timestamp
	u += "&seqid=3230596006172&ctype=huya_webh5&ver=1&fs=bgct&ratio=500&dMod=mseh-0&sdkPcdn=1_1&u=1470545821501&t=100&sv=2507070933&sdk_sid=1760687268998&a_block=0"
	log.Println(u)
	referer := "https://liveshare.huya.com/"
	origin := "https://liveshare.huya.com"
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"
	res, interval, cnt, err := HuyaRemoteAuth("111.199.230.32", origin, referer, ua, u)

	log.Println("Auth Result:", res, " RetryInterval:", interval, " RetryCnt:", cnt, " Err:", err)
	// 对url进行urlEncode编码
	encodedUrl := url.QueryEscape(u)

	sourceUrl := fmt.Sprintf("http://test-backsrc.huya.com/cdngw/backstreamurl/%s.flv?wsSecret=%s&wsTime=%s"+
		"&isp=cnc&prov=guangdong&city=guangzhou&clientip=45.23.33.2&requesturl=%s",
		conf.Stream, wsSecret, timestamp, encodedUrl)
	log.Println("sourceUrl:", sourceUrl)

	//hyP2pAuth(conf)
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

func HuyaP2pToken(streamName, wsTime string) string {
	splits := strings.Split(streamName, "_")
	streamSplit := ""
	if len(splits) > 0 {
		streamSplit = splits[0]
	}
	seed := "huya.com@live/pcdn/" + streamSplit + wsTime
	expectedSecret := getMd5Hash(seed)
	return expectedSecret
}

func HuyaRemoteAuth(userIp, origin, referer, userAgent, url string) (int, int, int, error) {
	// 构造请求体
	requestBody := fmt.Sprintf(`{
		"UserIp": "%s",
		"Origin": "%s",
		"Referer": "%s",
		"User-Agent": "%s",
		"URL": "%s"
	}`, userIp, origin, referer, userAgent, url)

	// 发送 HTTP POST 请求
	authURL := "http://urltoken.huya.com/urltoken/auth" // 测试地址
	resp, err := http.Post(authURL, "application/json", strings.NewReader(requestBody))
	if err != nil {
		log.Printf("发送远程鉴权请求失败: %v", err)
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("远程鉴权请求返回状态码异常: %d", resp.StatusCode)
		return 0, 0, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取远程鉴权响应体失败: %v", err)
		return 0, 0, 0, err
	}

	var response struct {
		AuthResult    int `json:"AuthResult"`
		RetryInterval int `json:"RetryInterval"`
		RetryCnt      int `json:"RetryCnt"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Printf("解析远程鉴权响应体失败: %v", err)
		return 0, 0, 0, err
	}

	log.Printf("远程鉴权结果: AuthResult=%d, RetryInterval=%d, RetryCnt=%d", response.AuthResult, response.RetryInterval, response.RetryCnt)
	return response.AuthResult, response.RetryInterval, response.RetryCnt, nil
}

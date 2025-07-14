package users

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"mikutool/config"
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
	t := time.Now().Unix()
	timestamp := strconv.FormatInt(t, 16)
	// 拼接固定部分和变动部分
	dataToHash := "huya.com@live/huyalive/" + conf.Stream + ".flv" + timestamp
	wsSecret := fmt.Sprintf("%x", dataToHash)

	wsSecret = getMd5Hash(wsSecret)

	// 生成请求地址
	u := "http://al.flv.huya.com/src/" + conf.Stream + ".flv?wsSecret=" + wsSecret + "&wsTime=" + timestamp
	log.Println(u)

	// 对url进行urlEncode编码
	encodedUrl := url.QueryEscape(u)

	sourceUrl := fmt.Sprintf("http://test-backsrc.huya.com/cdngw/backstreamurl/%s.flv?wsSecret=%s&wsTime=%s"+
		"&isp=cnc&prov=guangdong&city=guangzhou&clientip=45.23.33.2&requesturl=%s",
		conf.Stream, wsSecret, timestamp, encodedUrl)
	log.Println("sourceUrl:", sourceUrl)
}

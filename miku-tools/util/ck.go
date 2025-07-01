package util

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"middle-source-analysis/config"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/qbox/mikud-live/common/data"
)

var (
	CK driver.Conn
)

func NewCk(config *config.Config) driver.Conn {
	ckConf := config.CK
	ckOpts := &clickhouse.Options{
		Addr: ckConf.Host,
		Auth: clickhouse.Auth{
			Database: ckConf.DB,
			Username: ckConf.User,
			Password: ckConf.Passwd,
		},
		Debug:        false,
		DialTimeout:  time.Second * time.Duration(30),
		MaxOpenConns: 10,
		MaxIdleConns: 10,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},

		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10240,
	}
	conn, err := clickhouse.Open(ckOpts)
	if err != nil {
		panic(err)
	}
	return conn
}

func getMidnight() string {
	now := time.Now()

	// 获取当前日期的 0 点时间
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 格式化时间字符串，去掉时区信息
	midnightStr := midnight.Format("2006-01-02T15:04:05")
	return midnightStr
}

func GetMidnight2() string {
	now := time.Now()

	// 获取当前日期的 0 点时间
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 格式化时间字符串，去掉时区信息
	midnightStr := midnight.Format("2006-01-02 15:04:05")
	return midnightStr
}

func GetStreamNodeInfo(reqId, nodeId string) int64 {
	midnight := getMidnight()
	query := `
SELECT  CustomerSource, RequestID, StartTime, Status
FROM miku_data.streamd_qos
WHERE RequestID == '%s' AND Ts > '%s' AND NodeID == '%s'
LIMIT 1;
`
	query = fmt.Sprintf(query, reqId, midnight, nodeId)
	rows, err := CK.Query(context.Background(), query)
	if err != nil {
		log.Printf("query rows failed, err: %+v\n", err)
		return 0
	}
	defer rows.Close()
	for rows.Next() {
		var obj data.MikuQosObject
		if err := rows.ScanStruct(&obj); err != nil {
			log.Printf("rows ScanStruct failed, err: %+v\n", err)
			continue
		}
		return obj.StartTime
	}
	return 0
}

func GetIpByProvinceIsp(province, isp string) string {
	query := `
SELECT  Region, Isp, RemoteAddr
from miku_data.streamd_qos
WHERE Type = 'player' AND Region  = '%s' AND Isp = '%s'
LIMIT 1;
`
	query = fmt.Sprintf(query, province, isp)
	rows, err := CK.Query(context.Background(), query)
	if err != nil {
		log.Printf("query rows failed, err: %+v\n", err)
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var obj data.MikuQosObject
		if err := rows.ScanStruct(&obj); err != nil {
			log.Printf("rows ScanStruct failed, err: %+v\n", err)
			continue
		}
		parts := strings.Split(obj.RemoteAddr, ":")
		if len(parts) != 2 {
			log.Printf("parse remote addr err: %s\n", obj.RemoteAddr)
			continue
		}
		return parts[0]
	}
	return ""
}

func QueryCk(query string) []data.MikuQosObject {
	datas := make([]data.MikuQosObject, 0)
	rows, err := CK.Query(context.Background(), query)
	if err != nil {
		log.Printf("query rows failed, err: %+v\n", err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var obj data.MikuQosObject
		if err := rows.ScanStruct(&obj); err != nil {
			log.Printf("rows ScanStruct failed, err: %+v\n", err)
			continue
		}
		datas = append(datas, obj)
	}
	return datas
}

func Ck() {
	datas := QueryCk(Conf.Query)
	bytes, err := json.Marshal(datas)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/qbox/mikud-live/common/data"
)

func newCk(config *Config) driver.Conn {
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

func (s *Parser) GetStreamNodeInfo(reqId, nodeId string) int64 {
	midnight := getMidnight()
	query := `
SELECT  CustomerSource, RequestID, StartTime, Status
FROM miku_data.streamd_qos 
WHERE RequestID == '%s' AND Ts > '%s' AND NodeID == '%s'
LIMIT 1;
`
	query = fmt.Sprintf(query, reqId, midnight, nodeId)
	rows, err := s.ck.Query(context.Background(), query)
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

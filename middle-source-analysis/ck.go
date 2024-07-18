package main

import (
	"context"
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
		Debug:        true,
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

func (s *Parser) GetStreamNodeInfo() {
	query := `
SELECT  CustomerSource, RequestID, StartTime, NodeID, Status
FROM miku_data.streamd_qos 
WHERE RequestID == 'LhUNeIT1dVScoeIX' AND Ts > '2024-07-17T00:00:00' AND NodeID == 'd6c9c262-8d68-3378-a08a-a7acdf91548a-niulink64-site'
LIMIT 1;
`
	rows, err := s.ck.Query(context.Background(), query)
	if err != nil {
		log.Printf("query rows failed, err: %+v\n", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var obj data.MikuQosObject
		if err := rows.ScanStruct(&obj); err != nil {
			log.Printf("rows ScanStruct failed, err: %+v\n", err)
			continue
		}
		log.Printf("%+v\n", obj)
	}
}

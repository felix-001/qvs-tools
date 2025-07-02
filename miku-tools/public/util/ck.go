package util

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/qbox/mikud-live/common/data"
)

type Ck struct {
	conn driver.Conn
	conf *config.Config
}

func NewCk(config *config.Config) *Ck {
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
	ck := &Ck{
		conn: conn,
		conf: config,
	}
	return ck
}

func (c *Ck) QueryCk(query string) []data.MikuQosObject {
	datas := make([]data.MikuQosObject, 0)
	rows, err := c.conn.Query(context.Background(), query)
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

func (c *Ck) RunCk() {
	datas := c.QueryCk(c.conf.Query)
	bytes, err := json.Marshal(datas)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

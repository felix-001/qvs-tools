package qos

import (
	"encoding/json"
	"log"
)

type ChartMgr struct {
	charts []ChartConf
}

func NewChartMgr() *ChartMgr {
	return &ChartMgr{
		charts: []ChartConf{},
	}
}

func (c *ChartMgr) Parse() error {
	if err := json.Unmarshal([]byte(Charts_conf_json), &c.charts); err != nil {
		log.Printf("解析Charts_conf_json失败: %v\n", err)
		return err
	}
	/*
		for _, chart := range c.charts {
			sql, err := chart.buildSql()
			if err != nil {
				log.Printf("构建SQL失败: %v\n", err)
				return err
			}
			result, err := c.query(sql)
			if err != nil {
				log.Printf("查询失败: %v\n", err)
				return err
			}
		}
	*/
	return nil
}

package miku

import (
	"fmt"
	"log"
	"time"

	"github.com/qbox/mikud-live/cmd/lived/client"
)

type TingyunNodeChecker struct {
	conf TingyunConfig
}

type TingyunConfig struct {
	Enable            bool   `json:"enable"`
	NodeCheckInterval int    `json:"node_check_interval"`
	DetailId          string `json:"detail_id"` // 套餐id: 流媒体监测_2_pili
	AuthKey           string `json:"auth_key"`  // 听云鉴权key, 在听云web生成
	TaskType          int    `json:"task_type"` // 流媒体的taskType为3
	Domain            string `json:"domain"`    // 域名
}

type TaskConfig struct {
	Expire string `json:"expire"`
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Url    string `json:"url"`
}

func (t *TingyunNodeChecker) getValidTaskList() ([]TaskConfig, error) {
	log.Println("tingyun getValidTaskList")
	addr := fmt.Sprintf("https://%s/network-report-data/task-config/list-json-authkey?authkey=%s&detail_id=%s&status=1",
		t.conf.Domain, t.conf.AuthKey, t.conf.DetailId)
	resp := struct {
		TaskList []TaskConfig `json:"data"`
	}{}
	log.Println("tingyun getTaskList addr:", addr)
	httpCli := client.NewHttpCli()
	if err := httpCli.Request("GET", addr, "connid", nil, &resp); err != nil {
		log.Println("tingyun getTaskList failed", err)
		return nil, err
	}
	validTasks := make([]TaskConfig, 0)
	log.Println("tingyun getTaskList", resp.TaskList)
	for _, task := range resp.TaskList {
		expireTime, err := time.Parse("2006-01-02 15:04:05", task.Expire)
		if err != nil {
			log.Println("parse tingyun task expire time failed", err)
			continue
		}
		if time.Now().Before(expireTime) {
			validTasks = append(validTasks, task)
		} else {
			log.Println("tingyun task expired", task.Name, task.Expire)
		}
	}
	return validTasks, nil
}

func (m *Miku) getTaskList() ([]TaskConfig, error) {
	t := TingyunNodeChecker{}
	t.conf.AuthKey = m.conf.Key
	t.conf.DetailId = m.conf.ID
	t.conf.Domain = m.conf.Domain
	t.conf.TaskType = 3
	return t.getValidTaskList()
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var (
	uri = "http://qvs-pdr.qnlinking.com/api/v1/jobs"
)

type Pdr struct {
	Conf *Config
}

func NewPdr(conf *Config) *Pdr {
	return &Pdr{Conf: conf}
}

type QueryReq struct {
	Query     string `json:"query"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

func (s *Pdr) createJob(query string, start, end int64) (string, error) {
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": s.Conf.PdrToken,
	}

	q := QueryReq{
		Query:     query,
		StartTime: start,
		EndTime:   end,
	}
	//log.Println("q", q)
	body, err := json.Marshal(&q)
	if err != nil {
		log.Println(err)
		return "", err
	}
	resp, err := httpReq("POST", uri, string(body), headers)
	if err != nil {
		return "", err
	}
	result := &struct {
		ID string `json:"id"`
	}{}
	if err := json.Unmarshal([]byte(resp), result); err != nil {
		log.Println(err)
		return "", err
	}
	return result.ID, nil
}

func (s *Pdr) getJobProcess(jobId string) (int, error) {
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": s.Conf.PdrToken,
	}
	addr := fmt.Sprintf("%s/%s", uri, jobId)
	resp, err := httpReq("GET", addr, "", headers)
	if err != nil {
		return 0, err
	}
	result := &struct {
		Process int `json:"process"`
	}{}
	//log.Printf("result: %s\n", resp)
	if err := json.Unmarshal([]byte(resp), result); err != nil {
		log.Println(err)
		return 0, err
	}
	return result.Process, nil
}

func (s *Pdr) getRaw(jobId string) (string, error) {
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": s.Conf.PdrToken,
	}
	addr := fmt.Sprintf("%s/%s/events?rawLenLimit=false&pageSize=1000&order=desc&sort=updateTime", uri, jobId)
	resp, err := httpReq("GET", addr, "", headers)
	if err != nil {
		return "", err
	}
	return resp, err
}

type Host struct {
	Value string `json:"value"`
}

type Origin struct {
	Value string `json:"value"`
}

type Raw struct {
	Value string `json:"value"`
}

type Row struct {
	Raw    Raw    `json:"_raw"`
	Host   Host   `json:"host"`
	Origin Origin `json:"origin"`
}

type PdrLog struct {
	Total int   `json:"total"`
	Rows  []Row `json:"rows"`
}

func (s *Pdr) FetchLog(query string, start, end int64) (*PdrLog, error) {
	jobId, err := s.createJob(query, start, end)
	if err != nil {
		return nil, err
	}
	for {
		process, err := s.getJobProcess(jobId)
		if err != nil {
			return nil, err
		}
		if process == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	resp, err := s.getRaw(jobId)
	if err != nil {
		return nil, err
	}
	//log.Println(resp)
	pdrLog := PdrLog{}
	if err := json.Unmarshal([]byte(resp), &pdrLog); err != nil {
		log.Println(err)
		return nil, err
	}
	return &pdrLog, nil
}

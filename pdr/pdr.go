package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func partition(arr []string, low, high int) ([]string, int) {
	pivot := arr[high]
	i := low
	for j := low; j < high; j++ {
		if strings.Compare(arr[j], pivot) == -1 {
			arr[i], arr[j] = arr[j], arr[i]
			i++
		}
	}
	arr[i], arr[high] = arr[high], arr[i]
	return arr, i
}

func quickSort(arr []string, low, high int) []string {
	if low < high {
		var p int
		arr, p = partition(arr, low, high)
		arr = quickSort(arr, low, p-1)
		arr = quickSort(arr, p+1, high)
	}
	return arr
}

func getToken() (string, error) {
	b, err := ioutil.ReadFile("/usr/local/etc/pdr.conf")
	if err != nil {
		log.Println("read fail", "/usr/local/etc/pdr.conf", err)
		return "", err
	}
	return string(b), nil
}

type Pdr struct {
	token       string
	query       string
	collectSize int
	start       int64
	end         int64
	step        int64
}

type QueryData struct {
	StartTime   int64  `json:"startTime"`
	EndTime     int64  `json:"endTime"`
	Query       string `json:"query"`
	CollectSize int    `json:"collectSize"`
}

func httpReq(method, addr, body string, headers map[string]string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, addr, bytes.NewBuffer([]byte(body)))
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		log.Println("status code", resp.StatusCode, string(respBody))
		return nil, errors.New("http status code err")
	}
	return respBody, err
}

func (self *Pdr) createJob(start, end int64) (string, error) {
	addr := "http://qvs-pdr.qnlinking.com/api/v1/jobs"
	queryData := &QueryData{
		StartTime:   start,
		EndTime:     end,
		Query:       self.query,
		CollectSize: 500000,
	}
	data, err := json.Marshal(queryData)
	if err != nil {
		log.Println(err)
		return "", err
	}
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": self.token,
	}
	respBody, err := httpReq("POST", addr, string(data), headers)
	if err != nil {
		return "", err
	}
	res := &struct {
		Id string `json:"id"`
	}{}
	if err = json.Unmarshal(respBody, res); err != nil {
		log.Println(err)
		return "", err
	}
	return res.Id, err
}

func (self *Pdr) isJobDone(jobId string) (bool, error) {
	addr := "http://qvs-pdr.qnlinking.com/api/v1/jobs/" + jobId
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": self.token,
	}
	respBody, err := httpReq("GET", addr, "", headers)
	if err != nil {
		return false, err
	}
	res := &struct {
		Process int `json:"process"`
	}{}
	if err := json.Unmarshal(respBody, res); err != nil {
		return false, err
	}
	return res.Process == 1, nil
}

func (self *Pdr) waitJobDone(jobId string) {
	for {
		if done, err := self.isJobDone(jobId); err != nil && done {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (self *Pdr) pdrGet(addr string) ([]byte, error) {
	headers := map[string]string{
		"content-type":  "application/json",
		"Authorization": self.token,
	}
	return httpReq("GET", addr, "", headers)
}

func (self *Pdr) downloadLog(jobId string) (string, error) {
	addr := "http://qvs-pdr.qnlinking.com/api/v1/jobs/" + jobId + "/events?rawLenLimit=false&pageSize=5000000&prefix=&order=desc&sort=updateTime"
	respBody, err := self.pdrGet(addr)
	if err != nil {
		return "", err
	}
	res := &struct {
		Rows []struct {
			Raw struct {
				Value string `json:"value"`
			} `json:"_raw"`
		} `json:"rows"`
	}{}
	if err := json.Unmarshal(respBody, res); err != nil {
		return "", err
	}
	raw := ""
	for _, row := range res.Rows {
		raw += row.Raw.Value + "\n"
	}
	return raw, nil
}

func (self *Pdr) downloadAllLogs() (string, error) {
	logs := ""
	max := (self.end - self.start) / self.step
	for i := 0; i < int(max)-1; i++ {
		jobId, err := self.createJob(self.start+int64(i)*self.step, self.end+(int64(i)+1)*self.step)
		if err != nil {
			return "", err
		}
		self.waitJobDone(jobId)
		raw, err := self.downloadLog(jobId)
		if err != nil {
			return "", err
		}
		logs += raw
	}
	return logs, nil
}

func (self *Pdr) logFilter(logs string) []string {
	lines := strings.Split(logs, "\n")
	validLogs := []string{}
	for _, line := range lines {
		if len(line) > 0 && line[0] == '[' {
			validLogs = append(validLogs, line)
		}
	}
	return validLogs
}

func NewPdr(start, end, step int64, query, token string) *Pdr {
	if step == 0 {
		step = 10
	}
	return &Pdr{start: start, end: end, step: step, query: query, token: token}
}

func main() {
	start := flag.Int64("start", 0, "start time")
	end := flag.Int64("end", 0, "end time")
	step := flag.Int64("step", 0, "step")
	query := flag.String("query", "", "query")
	output := flag.String("output", "/tmp/pdr.log", "output log")
	flag.Parse()
	if *start == 0 || *end == 0 || *query == "" {
		flag.PrintDefaults()
		return
	}
	token, err := getToken()
	if err != nil {
		return
	}
	pdr := NewPdr(*start, *end, *step, *query, token)
	logs, err := pdr.downloadAllLogs()
	if err != nil {
		log.Println(err)
		return
	}
	logsFiltered := pdr.logFilter(logs)
	res := quickSort(logsFiltered, 0, len(logsFiltered)-1)
	txt := ""
	for _, log := range res {
		txt += log + "\n"
	}
	err = ioutil.WriteFile(*output, []byte(txt), 0644)
	if err != nil {
		log.Println(err)
		return
	}
}

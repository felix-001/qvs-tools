package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func httpGet(addr string) error {
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request : %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Request failed with status: %d\n", resp.StatusCode)
	}
	return nil
}

func getPCDN(addr string, cnt int) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		if latency > 1000 {
			fmt.Printf("%d latency: %d\n", cnt, latency)
		}
		if latency > 300 {
			fmt.Println(cnt, "cost", latency, "ms")
		}
	}()
	httpGet(addr)
}

var addr = "http://miku-lived-test.qiniuapi.com/app/stream01.xs?wsSecret=99550655ee7213d09e5cc797c00aac71&wsTime=68f50b00&did=123456"

//var addr = "http://miku-lived.qiniuapi.com/app/stream01.xs?wsSecret=99550655ee7213d09e5cc797c00aac71&wsTime=68f50b00&did=123456"

var wg sync.WaitGroup

func loop(addr string, cnt int) {
	defer wg.Done()
	for i := 0; i < cnt; i++ {
		getPCDN(addr, cnt)
	}
}

func main() {
	cnt := flag.Int("cnt", 100, "执行次数")
	flag.Parse()

	max := 300

	for i := 0; i < max; i++ {
		go loop(addr, *cnt)
	}
	wg.Add(max)
	wg.Wait()
}

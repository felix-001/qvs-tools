package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func main() {
	// 创建Elasticsearch客户端
	cfg := elasticsearch.Config{
		Addresses: []string{"http://10.60.35.22:9200"},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	// 构建搜索请求
	var (
		//indexName = "qvs-applog-data-stream-*"
		//query = `{"query": {"match": {"message": "OK"}}}`
		query = `{"query": {"match_all": {}}}`
	)

	req := esapi.SearchRequest{
		//Index: []string{indexName},
		Body: strings.NewReader(query),
	}

	// 发送搜索请求
	res, err := req.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("Error executing the search request: %s", err)
	}
	defer res.Body.Close()

	// 解析搜索结果
	if res.IsError() {
		log.Fatalf("Search request returned an error: %s", res.String())
	}

	fmt.Println(res.String())
}

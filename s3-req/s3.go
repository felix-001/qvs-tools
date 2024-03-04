package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Info struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// 定义要返回的数组
var sampleArray = []string{"aaa", "bbb", "ccc"}

// 设置CORS响应头
func setCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*") // 允许所有源
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// 定义处理函数
func handleRequest(w http.ResponseWriter, r *http.Request) {
	setCORS(&w)
	// 设置响应内容类型为JSON
	w.Header().Set("Content-Type", "application/json")

	ids := []Info{
		{ID: "aaa", Name: "aaa"},
		{ID: "bbb", Name: "bbb"},
		{ID: "ccc", Name: "ccc"},
	}
	// 将数组编码为JSON
	jsonArray, err := json.Marshal(ids)
	if err != nil {
		// 如果编码失败，返回错误信息
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 发送JSON响应
	w.Write(jsonArray)
}

func main() {
	// 设置HTTP服务器的监听地址
	http.HandleFunc("/", handleRequest)

	// 启动HTTP服务器，监听本地8080端口
	log.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

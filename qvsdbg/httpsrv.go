package main

import (
	"fmt"
	"log"
	"net/http"
)

func HttpSrvRun() {
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		// 从前端请求中读取数据
		data := r.FormValue("data")

		// 在后端进行处理，这里简单地将数据加上前缀
		response := "Processed: " + data

		// 将处理后的数据作为响应发送给前端
		fmt.Fprintf(w, response)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 将处理后的数据作为响应发送给前端
		fmt.Fprintf(w, html)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

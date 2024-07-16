package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *NetprobeSrv) initRouter(routers []Router, router *mux.Router) {
	for _, r := range routers {
		commonHandler := func(w http.ResponseWriter, req *http.Request) {
			log.Println(req)
			var handler func(paramMap map[string]string) string
			var params *[]string
			for _, r := range routers {
				if r.Path == req.URL.Path {
					log.Println(r.Path)
					handler = r.Handler
					params = &r.Params
					break
				}
			}
			paramMap := map[string]string{}
			for _, param := range *params {
				val := req.URL.Query().Get(param)
				paramMap[param] = val
			}
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusBadRequest)
				return
			}
			defer req.Body.Close()
			fmt.Println("Request Body:", string(body))
			paramMap["body"] = string(body)
			res := handler(paramMap)
			fmt.Fprintln(w, res)
		}
		router.HandleFunc(r.Path, commonHandler)
	}
}

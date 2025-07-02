package mock

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"regexp"

	"github.com/qbox/mikud-live/common/model"
)

type routeInfo struct {
	path    string
	handler http.HandlerFunc
}

func streamReport(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println(err, req.URL.String())
	}
	log.Println("body", string(body), req.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := model.StreamReportResponse{}
	bytes, err := json.Marshal(resp)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintln(w, string(bytes))
}

func pathQuery(w http.ResponseWriter, req *http.Request) {
	log.Println(req.URL.String())
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println(err, req.URL.String())
	}
	log.Println("pathquery body", string(body), req.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := model.PathQueryResponse{
		Code:    "1000",
		Message: "Success",
		Sources: []*model.SourceItem{
			{
				Node: "miku-source-player.cloudvdn.com",
				//Url:            "http://111.31.48.42:1370/bj/31011500991320021951.m3u8",
				//Url:            "https://miku-source-player.cloudvdn.com/youin-saas/live163098.m3u8?sign=c2c90410f26c08cebccf0c99c0b8006c&t=68be597e",
				//Url:            "http://1864314922.cloudvdn.com:1240/a.m3u8?domain=111.31.48.42&player=jkgAAF8fKR5hNfMX&secondToken=secondToken:ltxuclSWfHCbT_oZ0FN-F0DadQ8&streamid=bj:bj:bj/31011500991320021951&v3=1",
				Url: "https://miku-source-player.cloudvdn.com/youin-saas/live163372.m3u8?sign=7b6c448b29486298e26bfe01c28207ed&t=68beec73",
				//CustomerSource: true,
			},
		},
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintln(w, string(bytes))
}

func playcheck(w http.ResponseWriter, req *http.Request) {
	log.Println(req.URL.String())
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println(err, req.URL.String())
	}
	log.Println("playcheck body", string(body), req.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := model.StreamPlayResponse{
		Uid:       123456,
		ErrCode:   "1000",
		Message:   "Success",
		Bucket:    "app",
		Key:       "test",
		ConnectId: "connId",
		//FlowMethod: 2,
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintln(w, string(bytes))
}

func publishcheck(w http.ResponseWriter, req *http.Request) {
}

func publishdone(w http.ResponseWriter, req *http.Request) {
}

func pdrPoints(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println(err, req.URL.String())
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("pdrPonits body", string(body), req.URL.String())
}

var routePath = []routeInfo{
	{path: "/api/v1/streamreport", handler: streamReport},
	{path: "/api/v1/pathquery", handler: pathQuery},
	{path: "/api/v1/playcheck", handler: playcheck},
	{path: "/api/v1/publishcheck", handler: publishcheck},
	{path: "/api/v1/publishdone", handler: publishdone},
	{path: "/v4/repos/pili_vdn_streamd/points", handler: pdrPoints},
}

func route() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("url:", r.URL.Path)
		for _, route := range routePath {
			ok, err := regexp.Match(route.path, []byte(r.URL.Path))
			if err != nil {
				fmt.Println(err.Error())
			}
			if ok {
				route.handler(w, r)
				return
			}
		}
		if _, err := w.Write([]byte("404 not found")); err != nil {
			log.Fatal(err)
		}
	}
}

func MockAgent() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conn, err := net.Listen("tcp", "127.0.0.1:29991")
	if err != nil {
		log.Fatal(err)
	}
	if err := http.Serve(conn, route()); err != nil {
		log.Fatal(err)
	}
}

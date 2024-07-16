package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *NetprobeSrv) router() *mux.Router {
	getAreaIspNodesHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["areaIsp"]
		nodes := app.GetAreaIspNodesInfo(areaIsp)
		jsonbody, err := json.MarshalIndent(nodes, "", "    ")
		if err != nil {
			log.Println(err)
		}
		fmt.Fprint(w, string(jsonbody))
	}
	getAreaIspRootBwInfoHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["areaIsp"]
		info := app.GetAreaIspRootBwInfo(areaIsp)
		fmt.Fprint(w, info)
	}

	getAreaInfoHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["areaIsp"]
		info := app.GetAreaInfo(areaIsp)
		fmt.Fprint(w, info)
	}

	clearBwHandler := func(w http.ResponseWriter, req *http.Request) {
		nodeId := mux.Vars(req)["id"]
		app.ClearBw(nodeId)
		fmt.Fprint(w, "success")
	}

	fillAreaBwHandler := func(w http.ResponseWriter, req *http.Request) {
		areaIsp := mux.Vars(req)["area"]
		app.FillAreaBw(areaIsp)
		fmt.Fprint(w, "success")
	}
	fillIspBwHandler := func(w http.ResponseWriter, req *http.Request) {
		isp := mux.Vars(req)["isp"]
		app.FillIspBw(isp)
		fmt.Fprintf(w, "success")
	}
	router := mux.NewRouter()
	router.HandleFunc("/nodes/{areaIsp}", getAreaIspNodesHandler)
	router.HandleFunc("/area/{areaIsp}/rootBwInfo", getAreaIspRootBwInfoHandler)
	router.HandleFunc("/node/{id}/clearbw", clearBwHandler)
	router.HandleFunc("/area/{area}/fillAreaBw", fillAreaBwHandler)
	router.HandleFunc("/isp/{isp}/fillIspBw", fillIspBwHandler)
	router.HandleFunc("/area/{areaIsp}", getAreaInfoHandler)
	return router
}

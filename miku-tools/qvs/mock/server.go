package mock

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mikutool/miku/mock"
	"net"
	"net/http"
	"regexp"
)

var (
	uid              int
	recordFileFormat int
	help             bool
	recording        bool
	seqSnap          bool
	overwriteSnap    bool
)

func sipAuth(w http.ResponseWriter, req *http.Request) {
	log.Println("sip auth")
	fmt.Fprintln(w, "{\"code\":200}")
}

func getDevice(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "{\"state\":\"online\"}")
}

type TemplateInfo struct {
	Uid int `json:"uid"`

	RecordFileFormat      int    `json:"recordFileFormat"`
	RecordBucket          string `json:"recordBucket"`
	RecordType            int    `json:"recordType"`
	RecordNotifyURL       string `json:"recordNotifyURL"`
	RecordFileType        int    `json:"recordFileType"` // 存储方式, 0:普通存储,1:低频存储
	RecordDeleteAfterDays int    `json:"recordDeleteAfterdays"`
	TsFileNameTemplate    string `json:"tsFileNameTemplate"`
	Recording             bool   `json:"recording"`

	SnapBucket                   string `json:"snapBucket"`
	SnapDeleteAfterDays          int    `json:"snapDeleteAfterDays"`
	SnapFileType                 int    `json:"snapFileType"` // 存储方式, 0:普通存储,1:低频存储
	JpgSequenceStatus            bool   `json:"jpgSequenceStatus"`
	JpgOnDemandStatus            bool   `json:"jpgOnDemandStatus"` // 是否开启按需截图
	JpgOverwriteFileNameTemplate string `json:"jpgOverwriteFileNameTemplate"`
	JpgSequenceFileNameTemplate  string `json:"jpgSequenceFileNameTemplate"`
	JpgOnDemandFileNameTemplate  string `json:"jpgOnDemandFileNameTemplate"`

	RecordInterval int  `json:"recordInterval"`
	TsInterval     int  `json:"tsInterval"`
	SnapInterval   int  `json:"snapInterval"`
	SdcardSave     bool `json:"sdcardsave"`

	CallBack       string `json:"callback"`
	FirstCreatedAt int    `json:"firstCreatedAt"`

	CrossDay bool `json:"crossDay"`

	IsLNode              bool   `json:"isLNode" bson:"isLNode"`
	UseBaiduStreamFormat bool   `json:"useBaiduStreamFormat"`
	NamespaceAccessType  string `json:"namespaceAccessType" bson:"namespaceAccessType"`
}

func GetNamespaces_Streams_Template(w http.ResponseWriter, req *http.Request) {
	tmpl := &TemplateInfo{
		Uid:                         uid,
		TsFileNameTemplate:          "record/ts/${namespaceId}/${streamId}/${startMs}-${endMs}.ts",
		RecordType:                  1, // 录制模式，0（不录制），1（实时录制），2(按需录制)
		RecordFileFormat:            7,
		RecordBucket:                "liyqtest",
		RecordFileType:              0,
		RecordDeleteAfterDays:       0,
		Recording:                   recording,
		RecordInterval:              30, //录制文件时长 单位为秒，600~3600
		TsInterval:                  5,
		IsLNode:                     true,
		NamespaceAccessType:         "rtmp",
		SnapBucket:                  "liyqtest",
		SnapDeleteAfterDays:         7,
		SnapFileType:                0,
		JpgSequenceStatus:           seqSnap,
		JpgSequenceFileNameTemplate: "snapshot/jpg/${namespaceId}/${streamId}/${startMs}.jpg",
		SnapInterval:                3,
		JpgOnDemandFileNameTemplate: "snapshot/jpg/${namespaceId}/${streamId}/ondemand/${startMs}.jpg",
	}
	if overwriteSnap {
		tmpl.JpgOverwriteFileNameTemplate = "snapshot/jpg/${namespaceId}/${streamId}.jpg"
	}
	data, err := json.Marshal(tmpl)
	if err != nil {
		fmt.Fprintln(w, "{\"500\":\"internal err\"}")
		return
	}
	if _, err := fmt.Fprintln(w, string(data)); err != nil {
		fmt.Fprintln(w, "{\"500\":\"internal err\"}")
		return
	}
}

func GetSRSRtp(w http.ResponseWriter, req *http.Request) {
	fmt.Println("get /v1/srs/rtp")
	fmt.Fprintln(w, "{\"uid\":1}")
}

type routeInfo struct {
	path    string
	handler http.HandlerFunc
}

var routePaths = []routeInfo{
	//{path: "^/index/\\d+$", handler: index}, // \d: 匹配数字
	//{path: "^/home/\\w+$", handler: home},   // \w：匹配字母、数字、下划线
	{path: "/v1/devices/", handler: getDevice},
	{path: "/v1/srs/sip/auth", handler: sipAuth},
	{path: "/v1/namespaces/\\w+/streams/\\w+/template", handler: GetNamespaces_Streams_Template},
	{path: "/v1/srs/rtp", handler: GetSRSRtp},
}

func srvRoute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("url:", r.URL.Path)
		for _, route := range routePaths {
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

func MockSrv() {
	go MockThemisd()
	go MockTracker()
	go mock.MockLived()
	go MockHy()
	go Mockflume()
	go MockEcho()
	go MockM3u8()
	recording = false
	conn, err := net.Listen("tcp", "127.0.0.1:7275")
	if err != nil {
		log.Fatal(err)
	}
	if err := http.Serve(conn, srvRoute()); err != nil {
		log.Fatal(err)
	}
}

// 定义处理 /api/v1/getnodes POST 请求的函数
func getNodesHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if req.URL.Path != "/api/v1/getnodes" {
		http.NotFound(w, req)
		return
	}

	// 读取请求体
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	// 关闭请求体
	defer req.Body.Close()
	// 打印请求体内容
	fmt.Println(string(bodyBytes))

	// 定义要返回的数据
	responseData := map[string]interface{}{
		"code":    0,
		"message": "success",
		"addrs":   []string{"105.85.174.230:1234", "105.85.174.230:5678"},
	}

	// 将数据编码为 JSON
	jsonData, err := json.Marshal(responseData)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 写入响应数据
	if _, err := w.Write(jsonData); err != nil {
		log.Println("Failed to write response:", err)
	}
}

// 在 mockTracker 方法中启动一个新的 HTTP 服务器
func MockTracker() {
	http.HandleFunc("/api/v1/getnodes", getNodesHandler)
	go func() {
		if err := http.ListenAndServe("127.0.0.1:6008", nil); err != nil {
			log.Println("HTTP server error:", err)
		}
	}()
}

func MockHy() {
	http.HandleFunc("/cdngw/backstreamurl/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the dynamic part (xxx) from the URL
		path := req.URL.Path[len("/cdngw/backstreamurl/"):]
		if path == "" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Set the 302 redirect response
		http.Redirect(w, req, "http://124.236.65.37:1360/bj/31011500991180034748_31011500991320000091.flv", http.StatusFound)
	})

	go func() {
		log.Println("Starting Hy mock server on")
		if err := http.ListenAndServe("10.210.32.25:8008", nil); err != nil {
			log.Println("HTTP server error:", err)
		}
	}()
}

func Mockflume() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		fmt.Printf("Received request at path: %s\n", r.URL.Path)
		fmt.Printf("Request body: %s\n", string(body))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request received and logged"))
	})

	fmt.Println("Starting HTTP server on port 8085...")
	if err := http.ListenAndServe(":8085", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func MockEcho() {
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "your addr is: "+r.RemoteAddr)
	})

	err := http.ListenAndServe(":51370", nil)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func MockM3u8() {
	http.HandleFunc("/1016899595/823149915952238849/replay.1744797843.68854862.m3u8", func(w http.ResponseWriter, r *http.Request) {
		m3u8 := `
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-PLAYLIST-TYPE:VOD
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-TARGETDURATION:4
#EXT-X-DISCONTINUITY
#EXTINF:3.834,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1785624_3834_0_d1.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.903,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1337996_2903_1_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1284980_2902_2_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1289492_2902_3_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.903,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1270316_2903_4_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1284228_2902_5_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1302840_2902_6_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.903,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1273512_2903_7_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1249072_2902_8_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.926,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1295320_2926_9_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1254524_2902_10_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.903,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1155636_2903_11_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-775688_2902_12_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.902,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1347960_2902_13_d0.ts?qiniuvodi7kWcaHLk4=1
#EXTINF:2.694,
https://pull-hs3.vzan.com/1000926916/412514442310537988/112891393043-3-1244560_2694_14_d0.ts?qiniuvodi7kWcaHLk4=1
#EXT-X-ENDLIST
		`
		w.Header().Set("Content-Type", "application/x-mpegURL")

		w.Write([]byte(m3u8))
	})

	log.Println("Starting M3U8 mock server on port 8086...")
	if err := http.ListenAndServe(":8086", nil); err != nil {
		log.Printf("Server error: %v\n", err)
	}
}

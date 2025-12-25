package miku

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	_ "embed"
	"mikutool/config"
	"mikutool/miku/qos"
	_ "mikutool/miku/qos/hy"
	_ "mikutool/miku/qos/middle"
	_ "mikutool/miku/qos/upstream"
	_ "mikutool/miku/qos/usr"
	"mikutool/resources"
)

//go:embed qos/html/index.html
var indexHTML string

// QOSServer HTTP服务器结构
type QOSServer struct {
	config    *config.Config
	resources *resources.Resources
}

// NewQOSServer 创建新的QOS服务器
func NewQOSServer(cfg *config.Config, resources *resources.Resources) *QOSServer {
	return &QOSServer{
		config:    cfg,
		resources: resources,
	}
}

// StartServer 启动HTTP服务器
func (s *QOSServer) StartServer() {
	// 静态文件服务
	http.HandleFunc("/", s.homeHandler)

	// API接口
	http.HandleFunc("/api/v1/appnames", s.getAppNamesHandler)
	http.HandleFunc("/api/v1/qos", s.qosAnalysisHandler)
	http.HandleFunc("/api/v1/charts", s.chartsHandler)

	fmt.Println("QOS排障系统启动成功！")
	fmt.Println("请访问: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// homeHandler 主页面处理器
func (s *QOSServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// 返回HTML页面
	tmpl := template.Must(template.New("home").Parse(indexHTML))
	tmpl.Execute(w, nil)
}

// getAppNamesHandler 获取AppName列表处理器
func (s *QOSServer) getAppNamesHandler(w http.ResponseWriter, r *http.Request) {
	// 模拟AppName列表，实际应该从数据库获取
	appNames := []string{
		"huyacdn",
		"huyap2p",
		"livessports",
		"douyu",
		"vzan",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appNames)
}

// qosAnalysisHandler QOS分析处理器
func (s *QOSServer) qosAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req qos.QOSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("Error decoding request:", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	req.IpParser = s.resources.IpParser

	// Check and adjust time format if needed
	if strings.Count(req.StartTime, ":") == 1 {
		req.StartTime = req.StartTime + ":00"
	}
	if strings.Count(req.EndTime, ":") == 1 {
		req.EndTime = req.EndTime + ":00"
	}

	if req.LogLevel == "detail" {
		log.Printf("QOS analysis request: %+v", req)
	}

	if generator, ok := qos.ChartGenerators[req.Chart]; ok {
		data := generator.Generate(req)
		//log.Printf("data: %+v", data)
		bytes, err := json.Marshal(data)
		if err != nil {
			log.Println("Error marshalling chart data:", err)
			http.Error(w, "Error generating chart data", http.StatusInternalServerError)
			return
		}
		//log.Println("data", data)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

var chartsSeq = []string{
	// 用户
	"chart_onlineStreams",
	"chart_onlineUsers",
	"chart_usrDistributionCountry",
	"chart_usrDistributionArea",
	"chart_usrDistributionProv",
	"chart_lagUsrDistributionArea",
	"chart_lagUsrDistributionProv",
	"chart_lagrateByUser",
	"chart_hyLagUsrPercent",
	"chart_transcodeHyLagRateTrend",
	"chart_usrLagRate",
	"chart_HyTranscodeRateTrend",
	"chart_HyNormalRateTrend",
	// 边缘节点
	"chart_lagNodesPercent",
	"chart_hyLagrateByStreams",
	"chart_hyLagRateByNode",
	"chart_hyLagRateTrend",
	"chart_hy_nodeview_lag",
	// 中间源
	"chart_internalUpstreamLagRate",
	"chart_internalUpstreamRetryTimes",
	"chart_internalUpstreamRetryRate",
	// 源站
	"chart_customerUpstreamLagRate",
	"chart_customerUpstreamRetryTimes",
	"chart_upstreamAreaDis",
	"chart_upstreamBandwidth",
	"chart_videoFps",
	"chart_audioFps",
	"chart_HySrcLagRateTrend",
	"chart_HySrcRateTrend",
	"chart_lagSrcTranscodePercent",
	// 其他
	"chart_loadRate",
}

type ChartDataGetter interface {
	GetChartInfo() qos.ChartInfo
}

func (s *QOSServer) chartsHandler(w http.ResponseWriter, r *http.Request) {
	//log.Println("chartsHandler called")
	var charts []qos.ChartInfo
	for _, chart := range chartsSeq {
		//log.Println("chart", chart)
		if generator, ok := qos.ChartGenerators[chart]; ok {
			//log.Println("generator", generator)
			chartInfo := generator.ChartInfo()
			charts = append(charts, chartInfo)
		}
	}

	bytes, err := json.Marshal(charts)
	if err != nil {
		http.Error(w, "Error encoding chart data", http.StatusInternalServerError)
		return
	}

	log.Println("chartsHandler", string(bytes))
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

// Qos 主函数，启动QOS服务器
func Qos(config *config.Config, resources *resources.Resources) {
	if config == nil {
		//log.Fatal("配置不能为空")
	}

	server := NewQOSServer(config, resources)
	server.StartServer()
}

/*
回客户源站
    源站按大区分布
    回源帧率
    回源带宽
    回源重试次数 --
    回客户源站卡顿率
中间源
    内部回源重试次数
    内部回源重试率
    内部回源卡顿率
边缘节点
    边缘节点卡顿分析表格
    边缘节点卡顿聚集区域
    边缘节点卡顿聚集省份
用户
    用户卡顿分析表格
    用户按大区分布
    用户按省份分布
    用户卡顿聚集区域
    用户卡顿聚集省份
    流维度卡顿率
    协议维度卡顿率

    权重
    虎牙数据miku平替
    rtt
    重传率
*/

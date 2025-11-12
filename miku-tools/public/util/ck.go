package util

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Ck struct {
	conn driver.Conn
	conf *config.Config
}

func NewCk(config *config.Config) *Ck {
	ckConf := config.CK
	ckOpts := &clickhouse.Options{
		Addr: ckConf.Host,
		Auth: clickhouse.Auth{
			Database: ckConf.DB,
			Username: ckConf.User,
			Password: ckConf.Passwd,
		},
		Debug:        false,
		DialTimeout:  time.Second * time.Duration(30),
		MaxOpenConns: 10,
		MaxIdleConns: 10,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},

		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10240,
	}
	conn, err := clickhouse.Open(ckOpts)
	if err != nil {
		panic(err)
	}
	ck := &Ck{
		conn: conn,
		conf: config,
	}
	return ck
}

type Object struct {
	AppName            string    `json:"appName"`
	AudioFps           []float64 `json:"audioFps"`
	CustomerSource     bool      `json:"customerSource"`
	Domain             string    `json:"domain"`
	Fps                []float64 `json:"fps"`
	HitInfo            string    `json:"hitInfo"`
	HTTPReferer        string    `json:"httpReferer"`
	HTTPResponseCode   int64     `json:"httpResponseCode"`
	HTTPUserAgent      string    `json:"httpUserAgent"`
	IncomingBandwidth  []int64   `json:"incomingBandwidth"`
	LocalAddr          string    `json:"localAddr"`
	NodeID             string    `json:"nodeId"`
	OutgoingBandwidth  []int64   `json:"outgoingBandwidth"`
	Protocol           string    `json:"protocol"`
	RecvBytes          int64     `json:"recvBytes"`
	RemoteAddr         string    `json:"remoteAddr"`
	Request            string    `json:"request"`
	RequestID          string    `json:"requestId"`
	RequestTime        float64   `json:"requestTime"`
	SendBytes          int64     `json:"sendBytes"`
	StartTime          int64     `json:"startTime"`
	Status             string    `json:"status"`
	StreamName         string    `json:"streamName"`
	TotalRecvBytes     int64     `json:"totalRecvBytes"`
	TotalSendBytes     int64     `json:"totalSendBytes"`
	Timestamp          int64     `json:"ts"`
	Type               string    `json:"type"`
	UpstreamAddr       string    `json:"upstreamAddr"`
	UpstreamStartTime  int64     `json:"upstreamStartTime"`
	UpstreamURI        string    `json:"upstreamUri"`
	URL                string    `json:"url"`
	Ts                 time.Time `json:"-"`
	SendFirstPktTime   int64     `json:"sendFirstPktTime"`
	SendFirstAVPktTime uint64    `json:"sendFirstAVPktTime"`
	HttpLocation       string    `json:"httpLocation"`
	Uid                int64     `json:"uid"`
	Area               string    `json:"area"`

	Region            string  `json:"region"`
	City              string  `json:"city"`
	Isp               string  `json:"isp"`
	ErrorCode         int64   `json:"errorCode"`
	ErrorInfo         string  `json:"errorInfo"`
	ErrorDetail       string  `json:"errorDetail"`
	Latency           uint64  `json:"latency"`
	VideoCodec        string  `json:"videoCodec"`
	VideoProfile      string  `json:"videoProfile"`
	VideoRate         int64   `json:"videoRate"`
	VideoFps          float32 `json:"videoFps"`
	VideoResolution   string  `json:"videoResolution"`
	AudioCodec        string  `json:"audioCodec"`
	AudioProfile      string  `json:"audioProfile"`
	AudioRate         float32 `json:"audioRate"`
	AudioChannel      uint8   `json:"audioChannel"`
	AudioSample       uint32  `json:"audioSample"`
	LagDuration       int64   `json:"lagDuration"`
	LagCount          int64   `json:"lagCount"`
	LostVideoFrameCnt int64   `json:"lostVideoFrameCnt"`
	LostAudioFrameCnt int64   `json:"lostAudioFrameCnt"`
	InsertTs          int64   `json:"insertTs"`
	DropPuller        bool    `json:"dropPuller"`
	Version           string  `json:"version"`
	LatestDelay       int32   `json:"latestDelay"`
	LatestDouyuDelay  uint64  `json:"douyuDelay"`
	RetryTimes        uint32  `json:"retryTimes"`
	IpType            string  `json:"ipType"`
	CustomHeader      string  `json:"customHeader"`
	CostTime          int64   `json:"costTime"`
	UpstreamStatus    int64   `json:"upstreamStatus"`
	MetadataFps       float64 `json:"metadataFps"`
	ValidRate         float64 `json:"validRate"`
	Init              bool    `json:"init"`
	NodeArea          string  `json:"nodeArea"`
	NodeRegion        string  `json:"nodeRegion"`
	NodeCity          string  `json:"nodeCity"`
	NodeIsp           string  `json:"nodeIsp"`
	AvgGopSize        int64   `json:"avgGopSize"`
	FirstGopSentTime  int64   `json:"firstGopSentTime"`
	SourceStreamFps   float32 `json:"sourceStreamFps"`
	VideoDroppedRatio float64 `json:"videoDroppedRatio"`
	VideoGap          int64   `json:"videoGap"`
	CheckTime         int64   `json:"checkTime"`
	LatestSourceDelay int32   `json:"latestSourceDelay"` // 上报周期内最后一次延时统计(ms): 特指快手源站到边缘节点延时

	Rtt         []uint32  `json:"rtt"`
	RttVar      []uint32  `json:"rttVar"`
	LossRate    []float32 `json:"lossRate"`
	RetransRate []float32 `json:"retransRate"`
}

type CkCb func(rows driver.Rows)

func (c *Ck) QueryCk(query string, cb CkCb) error {
	//datas := make([]Object, 0)
	rows, err := c.conn.Query(context.Background(), query)
	if err != nil {
		log.Printf("query rows failed, err: %+v\n", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		cb(rows)
	}
	return nil
}

func (c *Ck) queryMikuQosData() []Object {
	var dests []Object
	cb := func(rows driver.Rows) {
		var dest Object
		if err := rows.ScanStruct(&dest); err != nil {
			log.Printf("rows ScanStruct failed, err: %+v\n", err)
			return
		}
		dests = append(dests, dest)
	}
	err := c.QueryCk(c.conf.Query, cb)
	if err != nil {
		log.Printf("query ck failed, err: %+v\n", err)
		return nil
	}
	bytes, err := json.Marshal(dests)
	if err != nil {
		log.Println(err)
		return nil
	}
	fmt.Println(string(bytes))
	return dests
}

func (c *Ck) RunCk() {
	results := c.queryMikuQosData()
	if results == nil {
		log.Println("no data to save")
		return
	}
	// 生成文件名：miku_qos_<日期>_<时间>.csv
	now := time.Now()
	fileName := fmt.Sprintf("/tmp/miku_qos_%s_%s.csv",
		now.Format("20060102"),
		now.Format("150405"),
	)
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("create csv file failed: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	header := []string{
		"appName", "customerSource", "domain", "hitInfo", "httpReferer", "httpResponseCode", "httpUserAgent",
		"localAddr", "nodeId", "protocol", "recvBytes", "remoteAddr", "request", "requestId", "requestTime",
		"sendBytes", "startTime", "status", "streamName", "totalRecvBytes", "totalSendBytes", "ts", "type",
		"upstreamAddr", "upstreamStartTime", "upstreamUri", "url", "sendFirstPktTime", "sendFirstAVPktTime",
		"httpLocation", "uid", "area", "region", "city", "isp", "errorCode", "errorInfo", "errorDetail",
		"latency", "videoCodec", "videoProfile", "videoRate", "videoFps", "videoResolution", "audioCodec",
		"audioProfile", "audioRate", "audioChannel", "audioSample", "lagDuration", "lagCount", "lostVideoFrameCnt",
		"lostAudioFrameCnt", "insertTs", "dropPuller", "version", "latestDelay", "douyuDelay", "retryTimes",
		"ipType", "customHeader", "costTime", "upstreamStatus", "metadataFps", "validRate", "init", "nodeArea",
		"nodeRegion", "nodeCity", "nodeIsp", "avgGopSize", "firstGopSentTime", "sourceStreamFps", "videoDroppedRatio",
		"videoGap", "checkTime", "latestSourceDelay", "audioFps", "fps", "incomingBandwidth", "outgoingBandwidth",
		"rtt", "rttVar", "lossRate", "retransRate",
	}
	if err := writer.Write(header); err != nil {
		log.Printf("write csv header failed: %v", err)
		return
	}

	// 写入数据行
	for _, r := range results {
		audioFps, _ := json.Marshal(r.AudioFps)
		fps, _ := json.Marshal(r.Fps)
		incomingBandwidth, _ := json.Marshal(r.IncomingBandwidth)
		outgoingBandwidth, _ := json.Marshal(r.OutgoingBandwidth)
		rtt, _ := json.Marshal(r.Rtt)
		rttVar, _ := json.Marshal(r.RttVar)
		lossRate, _ := json.Marshal(r.LossRate)
		retransRate, _ := json.Marshal(r.RetransRate)

		record := []string{
			r.AppName,
			fmt.Sprintf("%t", r.CustomerSource),
			r.Domain,
			r.HitInfo,
			r.HTTPReferer,
			fmt.Sprintf("%d", r.HTTPResponseCode),
			r.HTTPUserAgent,
			r.LocalAddr,
			r.NodeID,
			r.Protocol,
			fmt.Sprintf("%d", r.RecvBytes),
			r.RemoteAddr,
			r.Request,
			r.RequestID,
			fmt.Sprintf("%f", r.RequestTime),
			fmt.Sprintf("%d", r.SendBytes),
			fmt.Sprintf("%d", r.StartTime),
			r.Status,
			r.StreamName,
			fmt.Sprintf("%d", r.TotalRecvBytes),
			fmt.Sprintf("%d", r.TotalSendBytes),
			fmt.Sprintf("%d", r.Timestamp),
			r.Type,
			r.UpstreamAddr,
			fmt.Sprintf("%d", r.UpstreamStartTime),
			r.UpstreamURI,
			r.URL,
			fmt.Sprintf("%d", r.SendFirstPktTime),
			fmt.Sprintf("%d", r.SendFirstAVPktTime),
			r.HttpLocation,
			fmt.Sprintf("%d", r.Uid),
			r.Area,
			r.Region,
			r.City,
			r.Isp,
			fmt.Sprintf("%d", r.ErrorCode),
			r.ErrorInfo,
			r.ErrorDetail,
			fmt.Sprintf("%d", r.Latency),
			r.VideoCodec,
			r.VideoProfile,
			fmt.Sprintf("%d", r.VideoRate),
			fmt.Sprintf("%f", r.VideoFps),
			r.VideoResolution,
			r.AudioCodec,
			r.AudioProfile,
			fmt.Sprintf("%f", r.AudioRate),
			fmt.Sprintf("%d", r.AudioChannel),
			fmt.Sprintf("%d", r.AudioSample),
			fmt.Sprintf("%d", r.LagDuration),
			fmt.Sprintf("%d", r.LagCount),
			fmt.Sprintf("%d", r.LostVideoFrameCnt),
			fmt.Sprintf("%d", r.LostAudioFrameCnt),
			fmt.Sprintf("%d", r.InsertTs),
			fmt.Sprintf("%t", r.DropPuller),
			r.Version,
			fmt.Sprintf("%d", r.LatestDelay),
			fmt.Sprintf("%d", r.LatestDouyuDelay),
			fmt.Sprintf("%d", r.RetryTimes),
			r.IpType,
			r.CustomHeader,
			fmt.Sprintf("%d", r.CostTime),
			fmt.Sprintf("%d", r.UpstreamStatus),
			fmt.Sprintf("%f", r.MetadataFps),
			fmt.Sprintf("%f", r.ValidRate),
			fmt.Sprintf("%t", r.Init),
			r.NodeArea,
			r.NodeRegion,
			r.NodeCity,
			r.NodeIsp,
			fmt.Sprintf("%d", r.AvgGopSize),
			fmt.Sprintf("%d", r.FirstGopSentTime),
			fmt.Sprintf("%f", r.SourceStreamFps),
			fmt.Sprintf("%f", r.VideoDroppedRatio),
			fmt.Sprintf("%d", r.VideoGap),
			fmt.Sprintf("%d", r.CheckTime),
			fmt.Sprintf("%d", r.LatestSourceDelay),
			string(audioFps),
			string(fps),
			string(incomingBandwidth),
			string(outgoingBandwidth),
			string(rtt),
			string(rttVar),
			string(lossRate),
			string(retransRate),
		}
		if err := writer.Write(record); err != nil {
			log.Printf("write csv record failed: %v", err)
			return
		}
	}
	log.Printf("csv saved to %s", fileName)
}

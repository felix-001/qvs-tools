package util

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/trinodb/trino-go-client/trino"
)

type QualityReport struct {
	ClientType           *string `db:"client_type"`
	Cts                  *int64  `db:"cts"`
	DimIp                *string `db:"dim__ip"`
	DimIsp               *string `db:"dim__isp"`
	DimCdndomain         *string `db:"dim_cdndomain"`
	DimCdnip             *string `db:"dim_cdnip"`
	DimCoderatebps       *string `db:"dim_coderatebps"`
	DimHeartType         *string `db:"dim_heart_type"`
	DimIsInBackground    *string `db:"dim_is_in_background"`
	DimLine              *string `db:"dim_line"`
	DimNetworktype       *string `db:"dim_networktype"`
	DimPlatform          *string `db:"dim_platform"`
	DimStream            *string `db:"dim_stream"`
	DimStreamUrl         *string `db:"dim_stream_url"`
	DimVersion           *string `db:"dim_version"`
	FieldVideoBadQuality *int64  `db:"field_video_bad_quality"`
	InsertTs             *int64  `db:"insert_ts"`
	LogTime              *int64  `db:"log_time"`
	Systs                *int64  `db:"systs"`
	Minute               *string `db:"minute"`
	Innerreporttime      *int64  `db:"innerreporttime"`
	Innerfilepath        *string `db:"innerfilepath"`
	Day                  *string `db:"day"`
	Hour                 *string `db:"hour"`
}

type StreamdQosReport struct {
	Version            *string       `db:"version"`
	Type               *string       `db:"type"`
	Status             *string       `db:"status"`
	Errorcode          *int64        `db:"errorcode"`
	Errorinfo          *string       `db:"errorinfo"`
	Errordetail        *string       `db:"errordetail"`
	Nodeid             *string       `db:"nodeid"`
	Url                *string       `db:"url"`
	Request            *string       `db:"request"`
	Requestid          *string       `db:"requestid"`
	Protocol           *string       `db:"protocol"`
	Transprotocol      *string       `db:"transprotocol"`
	Scheme             *string       `db:"scheme"`
	Domain             *string       `db:"domain"`
	Uid                *int64        `db:"uid"`
	Appname            *string       `db:"appname"`
	Streamname         *string       `db:"streamname"`
	Localaddr          *string       `db:"localaddr"`
	Remoteaddr         *string       `db:"remoteaddr"`
	Httpreferer        *string       `db:"httpreferer"`
	Httpuseragent      *string       `db:"httpuseragent"`
	Hitinfo            *string       `db:"hitinfo"`
	Upstreamuri        *string       `db:"upstreamuri"`
	Upstreamaddr       *string       `db:"upstreamaddr"`
	Starttime          *int64        `db:"starttime"`
	Publisher          *bool         `db:"publisher"`
	Master             *bool         `db:"master"`
	Httpresponsecode   *int64        `db:"httpresponsecode"`
	Upstreamstarttime  *int64        `db:"upstreamstarttime"`
	Requesttime        *float64      `db:"requesttime"`
	Totalsendbytes     *int64        `db:"totalsendbytes"`
	Totalrecvbytes     *int64        `db:"totalrecvbytes"`
	Sendbytes          *int64        `db:"sendbytes"`
	Recvbytes          *int64        `db:"recvbytes"`
	Outgoingbandwidth  []interface{} `db:"outgoingbandwidth"`
	Incomingbandwidth  []interface{} `db:"incomingbandwidth"`
	Fps                []interface{} `db:"fps"`
	Audiofps           []interface{} `db:"audiofps"`
	Ts                 *int64        `db:"ts"`
	Customersource     *bool         `db:"customersource"`
	Sendfirstpkttime   *int64        `db:"sendfirstpkttime"`
	Httplocation       *string       `db:"httplocation"`
	Latency            *int64        `db:"latency"`
	Lagduration        *int64        `db:"lagduration"`
	Lagcount           *int64        `db:"lagcount"`
	Lostvideoframecnt  *int64        `db:"lostvideoframecnt"`
	Lostaudioframecnt  *int64        `db:"lostaudioframecnt"`
	Droppuller         *bool         `db:"droppuller"`
	Minute             *string       `db:"minute"`
	Innerreporttime    *int64        `db:"innerreporttime"`
	Innerfilepath      *string       `db:"innerfilepath"`
	Sendfirstavpkttime *int64        `db:"sendfirstavpkttime"`
	Latestsourcedelay  *int32        `db:"latestsourcedelay"`
	Area               *string       `db:"area"`
	Region             *string       `db:"region"`
	City               *string       `db:"city"`
	Retrytimes         *int64        `db:"retrytimes"`
	Costtime           *int64        `db:"costtime"`
	Nodearea           *string       `db:"nodearea"`
	Noderegion         *string       `db:"noderegion"`
	Nodecity           *string       `db:"nodecity"`
	Nodeisp            *string       `db:"nodeisp"`
	Checktime          *int64        `db:"checktime"`
	Rtt                []interface{} `db:"rtt"`
	Rttvar             []interface{} `db:"rttvar"`
	Retransrate        []interface{} `db:"retransrate"`
	Isp                *string       `db:"isp"`
	Machineid          *string       `db:"machineid"`
	Day                *string       `db:"day"`
	Hour               *string       `db:"hour"`
	Ts_m               *string       `db:"ts_m"`
}

type StreamdLagReport struct {
	Ts_m                               *string  `db:"ts_m"`
	Total_lag_duration_player          *float64 `db:"total_lag_duration_player"`
	Total_cost_time_player             *float64 `db:"total_cost_time_player"`
	Ratio_lag_player                   *float64 `db:"ratio_lag_player"`
	Hit_total_lag_duration_player      *float64 `db:"hit_total_lag_duration_player"`
	Hit_total_cost_time_player         *float64 `db:"hit_total_cost_time_player"`
	Hit_ratio_lag_player               *float64 `db:"hit_ratio_lag_player"`
	Not_hit_total_lag_duration_player  *float64 `db:"not_hit_total_lag_duration_player"`
	Not_hit_total_cost_time_player     *float64 `db:"not_hit_total_cost_time_player"`
	Not_hit_ratio_lag_player           *float64 `db:"not_hit_ratio_lag_player"`
	Total_lag_duration_puller          *float64 `db:"total_lag_duration_puller"`
	Total_cost_time_puller             *float64 `db:"total_cost_time_puller"`
	Ratio_lag_puller                   *float64 `db:"ratio_lag_puller"`
	Total_lag_duration_internal_player *float64 `db:"total_lag_duration_internal_player"`
	Total_cost_time_internal_player    *float64 `db:"total_cost_time_internal_player"`
	Ratio_lag_internal_player          *float64 `db:"ratio_lag_internal_player"`
	Total_lag_duration_publisher       *float64 `db:"total_lag_duration_publisher"`
	Total_cost_time_publisher          *float64 `db:"total_cost_time_publisher"`
	Ratio_lag_publisher                *float64 `db:"ratio_lag_publisher"`
	Requests_puller                    *int64   `db:"requests_puller"`
	Retry_requests_puller              *int64   `db:"retry_requests_puller"`
	Retry_ratio_puller                 *float64 `db:"retry_ratio_puller"`
	TotalRetryTimes                    *int64   `db:"totalRetryTimes"`
}

type StreamdFpsReport struct {
	Ts                   *string  `db:"ts"`
	NodeId               *string  `db:"NodeID"`
	StreamName           *string  `db:"StreamName"`
	AppName              *string  `db:"AppName"`
	Avg_IncomingVideoFps *float64 `db:"avg_IncomingVideoFps"`
	Avg_IncomingAudioFps *float64 `db:"avg_IncomingAudioFps"`
	SourceType           *string  `db:"source_type"`
}

type StreamdStreamCntReport struct {
	Ts_m      *string `db:"ts_m"`
	StreamCnt *int64  `db:"stream_cnt"`
}

type HyCdnLagReport struct {
	DimCdnip *string  `db:"dim_cdnip"`
	Percent  *float64 `db:"percent"`
	LagCnt   *int     `db:"lagCnt"`
	Total    *int     `db:"total"`
	DimIp    *string  `db:"dim__ip"`
}

type HyClientIpsOnCdnIpReport struct {
	DimCdnip     *string `db:"dim_cdnip"`
	DimIp        *string `db:"dim__ip"`
	DimStreamUrl *string `db:"dim_stream_url"` // 从url解析出streamName再做聚合，一个cdnip跑了哪些流
	LagCnt       *int    `db:"lagCnt"`
}

type StreamdUpstreamBandWidthReport struct {
	Ts         *string  `db:"ts"`
	NodeId     *string  `db:"NodeID"`
	StreamName *string  `db:"StreamName"`
	AppName    *string  `db:"AppName"`
	BandWidth  *float64 `db:"bandwidth"`
	SourceType *string  `db:"source_type"`
}

type UpstreamDistributeReport struct {
	RemoteAddr *string `db:"RemoteAddr"`
}

type HyLagReport struct {
	Ts_m         *string  `db:"ts_m"`
	Percent      *float64 `db:"percent"`
	LagCnt       *int     `db:"lagCnt"`
	Total        *int     `db:"total"`
	LagUsrCnt    *int     `db:"lag_usr_cnt"`
	TotalUsrCnt  *int     `db:"total_usr_cnt"`
	LagUsrRate   *float64 `db:"lag_usr_rate"`
	LagNodeCnt   *int     `db:"lag_node_cnt"`
	TotalNodeCnt *int     `db:"total_node_cnt"`
	LagNodeRate  *float64 `db:"lag_node_rate"`
}

type HyLagRateByStreamsReport struct {
	Ts_m       *string  `db:"ts_m"`
	StreamName *string  `db:"stream_id"`
	Percent    *float64 `db:"percent"`
	LagCnt     *int     `db:"lagCnt"`
	Total      *int     `db:"total"`
}

func TrinoQuery(schema, sql string, dest interface{}) error {
	dsn := fmt.Sprintf("http://superset@trino.jf-logverse.k8s.qiniu.io?catalog=hive_miku&schema=%s", schema)
	db, err := sqlx.Open("trino", dsn)
	if err != nil {
		log.Println(err)
		return err
	}
	defer db.Close()

	err = db.Select(dest, sql)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

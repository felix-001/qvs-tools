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
	LoadCnt                            *int64   `db:"loadCnt"`
	Total                              *int64   `db:"total"`
	LoadRatio                          *float64 `db:"loadRatio"`
	RetryRatioUpstream                 *float64 `db:"retry_ratio_upstream"`
	UpstreamRetryTimes                 *int64   `db:"upstreamRetryTimes"`
	UsrCnt                             *int64   `db:"usr_cnt"`
	Patch_fail_percent                 *float64 `db:"patch_fail_percent"`
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
	StreamName   *string `db:"streamName"`
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
	Ts_m                 *string  `db:"ts_m"`
	Percent              *float64 `db:"percent"`
	LagCnt               *int     `db:"lagCnt"`
	Total                *int     `db:"total"`
	LagUsrCnt            *int     `db:"lag_usr_cnt"`
	TotalUsrCnt          *int     `db:"total_usr_cnt"`
	LagUsrRate           *float64 `db:"lag_usr_rate"`
	LagNodeCnt           *int     `db:"lag_node_cnt"`
	TotalNodeCnt         *int     `db:"total_node_cnt"`
	LagNodeRate          *float64 `db:"lag_node_rate"`
	TranscodeLagCnt      *int     `db:"transcodeLagCnt"`      // 推流MIKU转码流卡顿样本个数
	TotalTranscodeCnt    *int     `db:"totalTranscodeCnt"`    // 总转码流样本个数
	NotTranscodeLagCnt   *int     `db:"notTranscodeLagCnt"`   // 推流MIKU非转码流卡顿样本个数
	TotalNotTranscodeCnt *int     `db:"totalNotTranscodeCnt"` // 总非转码流样本个数
	MikuNormalLagRate    *float64 `db:"mikuNormalLagRate"`    // 推流MIKU非转码流卡顿率
	MikuTrancodeLagRate  *float64 `db:"mikuTrancodeLagRate"`  // 推流MIKU转码流卡顿率
	SrcTranscodeLagRate  *float64 `db:"srcTranscodeLagRate"`  // 回客户源转码流卡顿率
	SrcNormalLagRate     *float64 `db:"srcNormalLagRate"`     // 回源客户源非转码流卡顿率
	SrcLagRate           *float64 `db:"srcLagRate"`           // 回源客户整体卡顿率
	MikuTranscodePercent *float64 `db:"mikuTranscodePercent"` // 推流MIKU转码流样本占总样本的百分比
	NormalRate           *float64 `db:"normalRate"`           // 推流MIKU非转码流样本占总样本的百分比
	SrcNormalPercent     *float64 `db:"srcNormalPercent"`     // 回源客户源非转码流样本占总样本的百分比
	SrcTranscodePercent  *float64 `db:"srcTranscodePercent"`  // 回源客户源转码流样本占总样本的百分比
	SrcRate              *float64 `db:"srcRate"`              // 回客户源站样本占总样本的百分比
	P2pPercent           *float64 `db:"p2p_percent"`          // p2p样本占总样本的比重
	PatchLagPercent      *float64 `db:"patch_lag_percent"`    // 虎牙补片卡顿样本占总p2p卡顿样本的比例
}

type HyLagRateByStreamsReport struct {
	Ts_m            *string  `db:"ts_m" json:"-"`
	StreamName      *string  `db:"stream_id"`
	Percent         *float64 `db:"percent"`
	LagCnt          *int     `db:"lagCnt"`
	Total           *int     `db:"total"`
	Weight          *float64 `db:"weight"`
	LagClientIps    []string `db:"lag_client_ips"`
	NormalClientIps []string `db:"normal_client_ips"`
	LagClientRate   float64  `db:"lag_client_rate"`
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

func TrinoQueryMap(schema, sql string, dest interface{}) error {
	dsn := fmt.Sprintf("http://superset@trino.jf-logverse.k8s.qiniu.io?catalog=hive_miku&schema=%s", schema)
	db, err := sqlx.Open("trino", dsn)
	if err != nil {
		log.Println(err)
		return err
	}
	defer db.Close()

	// 检查 dest 类型
	switch d := dest.(type) {
	case *[]map[string]any:
		// 对于 map 切片，使用特殊的处理方式
		rows, err := db.Query(sql)
		if err != nil {
			return err
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return err
		}

		*d = []map[string]any{}
		for rows.Next() {
			// 为每列创建值的切片
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return err
			}

			// 创建 map
			rowMap := make(map[string]any)
			for i, col := range columns {
				val := values[i]

				// 处理 byte 数组（通常是字符串）
				if b, ok := val.([]byte); ok {
					rowMap[col] = string(b)
				} else {
					rowMap[col] = val
				}
			}
			*d = append(*d, rowMap)
		}
		return rows.Err()
	default:
		// 其他类型（如结构体切片）使用原有的 Select
		err = db.Select(dest, sql)
		if err != nil {
			log.Println(err)
			return err
		}
		return nil
	}
}

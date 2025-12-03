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

func TrinoQuery(schema, sql string, dest interface{}) error {
	dsn := fmt.Sprintf("http://superset@trino.jf-logverse.k8s.qiniu.io?catalog=hive_miku&schema=%s", schema)
	db, err := sqlx.Open("trino", dsn)
	if err != nil {
		log.Println(err)
		return err
	}
	defer db.Close()

	/*
			err = db.Select(&reports, `
		        select *
		        from huyabiz_quality_report_log where day='20251203' limit 10`)
			if err != nil {
				log.Fatal(err)
			}
	*/

	err = db.Select(dest, sql)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

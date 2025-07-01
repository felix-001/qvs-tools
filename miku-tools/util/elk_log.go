package util

import (
	"log"
)

var LogColumn = 19

func LoadElkLog(filename string) (logs []string) {
	rows := LocadCsv(filename)
	columnCnt := len(rows[0])
	if columnCnt < LogColumn {
		log.Println("column cnt chk err")
		return nil
	}
	log.Println("columnCnt:", columnCnt)
	for _, row := range rows {
		if len(row) < columnCnt {
			log.Println("row column not enough", row)
			continue
		}
		logs = append(logs, row[LogColumn-1])
	}
	return
}

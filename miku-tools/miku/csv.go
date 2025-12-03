package miku

import (
	"encoding/csv"
	"log"
	"mikutool/public/util"
	"os"
)

func (s *QOSServer) saveCsv(reports []util.QualityReport) {
	// 将reports格式化为CSV并写入文件
	csvFile, err := os.Create("qos_report.csv")
	if err != nil {
		log.Printf("创建CSV文件失败: %v", err)
	} else {
		defer csvFile.Close()

		writer := csv.NewWriter(csvFile)
		defer writer.Flush()

		// 写入CSV头
		headers := []string{
			"ClientType", "Cts", "DimIp", "DimIsp", "DimCdndomain", "DimCdnip",
			"DimCoderatebps", "DimHeartType", "DimIsInBackground", "DimLine",
			"DimNetworktype", "DimPlatform", "DimStream", "DimStreamUrl", "DimVersion",
			"FieldVideoBadQuality", "InsertTs", "LogTime", "Systs", "Minute",
			"Innerreporttime", "Innerfilepath", "Day", "Hour",
		}
		if err := writer.Write(headers); err != nil {
			log.Printf("写入CSV头失败: %v", err)
		}

		// 写入数据行
		for _, report := range reports {
			record := []string{
				s.getString(report.ClientType),
				s.getInt64(report.Cts),
				s.getString(report.DimIp),
				s.getString(report.DimIsp),
				s.getString(report.DimCdndomain),
				s.getString(report.DimCdnip),
				s.getString(report.DimCoderatebps),
				s.getString(report.DimHeartType),
				s.getString(report.DimIsInBackground),
				s.getString(report.DimLine),
				s.getString(report.DimNetworktype),
				s.getString(report.DimPlatform),
				s.getString(report.DimStream),
				s.getString(report.DimStreamUrl),
				s.getString(report.DimVersion),
				s.getInt64(report.FieldVideoBadQuality),
				s.getInt64(report.InsertTs),
				s.getInt64(report.LogTime),
				s.getInt64(report.Systs),
				s.getString(report.Minute),
				s.getInt64(report.Innerreporttime),
				s.getString(report.Innerfilepath),
				s.getString(report.Day),
				s.getString(report.Hour),
			}
			if err := writer.Write(record); err != nil {
				log.Printf("写入CSV记录失败: %v", err)
			}
		}
		log.Println("CSV报告已保存到 qos_report.csv")
	}
}

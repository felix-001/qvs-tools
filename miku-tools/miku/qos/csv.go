package qos

import (
	"encoding/csv"
	"log"
	"mikutool/public/util"
	"os"
)

func saveCsv(reports []util.QualityReport) {
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
				getString(report.ClientType),
				getInt64(report.Cts),
				getString(report.DimIp),
				getString(report.DimIsp),
				getString(report.DimCdndomain),
				getString(report.DimCdnip),
				getString(report.DimCoderatebps),
				getString(report.DimHeartType),
				getString(report.DimIsInBackground),
				getString(report.DimLine),
				getString(report.DimNetworktype),
				getString(report.DimPlatform),
				getString(report.DimStream),
				getString(report.DimStreamUrl),
				getString(report.DimVersion),
				getInt64(report.FieldVideoBadQuality),
				getInt64(report.InsertTs),
				getInt64(report.LogTime),
				getInt64(report.Systs),
				getString(report.Minute),
				getInt64(report.Innerreporttime),
				getString(report.Innerfilepath),
				getString(report.Day),
				getString(report.Hour),
			}
			if err := writer.Write(record); err != nil {
				log.Printf("写入CSV记录失败: %v", err)
			}
		}
		log.Println("CSV报告已保存到 qos_report.csv")
	}
}

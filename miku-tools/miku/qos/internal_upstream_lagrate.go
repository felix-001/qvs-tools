package qos

import (
	"fmt"
	"log"
)

// 内部回源百秒卡顿率趋势图

func init() {
	RegisterChartGenerator("InternalUpstreamLagRate", &InternalUpstreamLagRate{})
}

type InternalUpstreamLagRate struct {
}

func (i *InternalUpstreamLagRate) Generate(req QOSRequest) any {
	if !req.Charts.InternalUpstreamLagRate {
		return ""
	}
	streamdReports, err := GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	/*
		data := LineChartData{
			Title:       "内部回源百秒卡顿率",
			SeriesTitle: "卡顿率",
			Color:       "#1890ff",
			XType:       "category",
			XAxis:       []string{},
			YAxis:       []string{},
		}
	*/
	log.Println("查询结果streamdReports:", len(streamdReports))
	html := fmt.Sprintf(`
	<div style="margin-top: 20px;">
		<div style="margin-bottom: 30px;">
			<h4>内部回源百秒卡顿率趋势图</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
	</div>`,
		generateLineChartHTML(streamdReports, "Ts_m", "内部回源百秒卡顿率", "内部回源百秒卡顿率"),
	)
	return html
}

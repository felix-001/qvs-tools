package qos

import (
	"fmt"
	"log"
)

// 内部回源重试率趋势图

func init() {
	RegisterChartGenerator("InternalUpstreamRetryRate", &InternalUpstreamRetryRate{})
}

type InternalUpstreamRetryRate struct {
}

func (i *InternalUpstreamRetryRate) Generate(req QOSRequest) any {
	if !req.Charts.InternalUpstreamRetryRate {
		return ""
	}
	streamdReports, err := GetMikuStreamdReportLagDatas(req)
	if err != nil {
		log.Println("获取数据失败:", err)
		return ""
	}
	log.Println("查询结果streamdReports:", len(streamdReports))
	html := fmt.Sprintf(`
	<div style="margin-top: 20px;">
		<div style="margin-bottom: 30px;">
			<h4>内部回源重试率趋势图</h4>
			<iframe src="%s" width="100%%" height="300" frameborder="0" style="border: 1px solid #ddd; border-radius: 4px;"></iframe>
		</div>
	</div>
`, generateLineChartHTML(streamdReports, "Ts_m", "内部回源重试率", "内部回源重试率"))
	return html
}

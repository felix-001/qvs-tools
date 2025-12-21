package hy

import (
	"fmt"
	"log"
	"mikutool/miku/qos"
	"mikutool/public/util"
)

// 指定节点ip，查看节点的每分钟卡顿率趋势

func init() {
	qos.RegisterChartGenerator(&HyNodeViewLag{})
}

type HyNodeViewLag struct {
}

func (h *HyNodeViewLag) Generate(req qos.QOSRequest) any {
	if req.CdnIp == "" {
		return qos.ChartData{Data: "", Type: "error"}
	}
	sql := qos.BuildHyCdnIpLagSQLQuery(req)
	if req.LogLevel == "detail" {
		log.Printf("执行SQL查询: %s", sql)
	}
	var hyCdnIpLagReports []util.HyLagReport
	if err := util.TrinoQuery("miku", sql, &hyCdnIpLagReports); err != nil {
		log.Printf("Trino查询失败: %v", err)
		return qos.ChartData{Data: fmt.Sprintf("查询失败: %v", err), Type: "error"}
	}
	log.Println("查询结果hyCdnIpLagReports:", len(hyCdnIpLagReports))

	fn := func(cb qos.Cb) {
		for _, report := range hyCdnIpLagReports {
			if report.Percent == nil {
				continue
			}
			cb(*report.Ts_m, *report.Percent)
		}
	}
	chartData := qos.GenerateLineChartData("节点卡顿率趋势", "卡顿率", fn)
	return qos.ChartData{Data: chartData, Type: qos.ChartTypeLine}
}

func (h *HyNodeViewLag) ID() string {
	return "chart_hy_nodeview_lag"
}

func (h *HyNodeViewLag) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    h.ID(),
		Title: "虎牙卡顿节点占比趋势图",
	}
}

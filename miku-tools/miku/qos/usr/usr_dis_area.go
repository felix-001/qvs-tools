package usr

import (
	"log"
	"mikutool/miku/qos"
)

func init() {
	log.Println("init usr_dis_area")
	qos.RegisterChartGenerator(&UserDistributionArea{
		Title: "用户分布区域(MIKU)",
	})
}

type UserDistributionArea struct {
	Title string
}

func (u *UserDistributionArea) Generate(req qos.QOSRequest) any {
	// 这里应该返回实际的用户分布数据
	// 示例数据，实际应该从数据库查询
	data := []map[string]interface{}{
		{"name": "华东", "value": 53},
		{"name": "华中", "value": 34},
		{"name": "华南", "value": 11},
		{"name": "东北", "value": 6},
		{"name": "华北", "value": 5},
		{"name": "其他", "value": 2},
	}

	return map[string]interface{}{
		"title": u.Title,
		"data":  data,
	}
}
func (u *UserDistributionArea) ChartInfo() qos.ChartInfo {
	return qos.ChartInfo{
		ID:    u.ID(),
		Title: u.Title,
	}
}

func (u *UserDistributionArea) ID() string {
	return "chart_usrDistributionAreaMiku"
}

package staging

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"
)

type MikuStagMgr struct {
}

func NewMikuStagMgr() *MikuStagMgr {
	return &MikuStagMgr{}
}

type Device struct {
	Gbid              string `json:"gbId"`
	AlarmTypesForSnap string `json:"alarmTypesForSnap"`
}

type Devices struct {
	Items []Device `json:"items"`
	Total int      `json:"total"`
}

func TestDev(config *config.Config) {
	processed := 0
	for i := 1; i < 1000; i++ {
		config.Method = "GET"
		config.Addr = fmt.Sprintf("http://qvs.qiniuapi.com/v1/namespaces/%s/devices?line=500&offset=%d",
			config.App, (i-1)*500)
		config.Body = ""
		resp, err := util.HttpRequest(config)
		if err != nil {
			log.Println("http request failed:", err)
			return
		}

		var devices Devices
		if err := json.Unmarshal([]byte(resp), &devices); err != nil {
			log.Println("json unmarshal failed:", err)
			return
		}

		fmt.Println("Response:", len(devices.Items))
		for _, device := range devices.Items {
			if device.AlarmTypesForSnap != "" {
				fmt.Println(device)
				config.Method = "PATCH"
				config.Addr = fmt.Sprintf("http://qvs.qiniuapi.com/v1/namespaces/%s/devices/%s",
					config.App, device.Gbid)
				config.Body = `{"operations":[{
							"key":"alarmTypesForSnap",
							"op":"replace",
							"value":"" 
						}]}`
				_, err := util.HttpRequest(config)
				if err != nil {
					log.Println("http request failed:", err)
					return
				}
				processed++
				log.Println("Processed:", processed)
				//return
				//if processed >= 10 {
				//	return
				//}
			}
		}
		log.Println("Processing batch:", i, devices.Total/500)
		//log.Println("total:", devices.Total)
		if (i-1)*500 >= devices.Total {
			break
		}
	}
}

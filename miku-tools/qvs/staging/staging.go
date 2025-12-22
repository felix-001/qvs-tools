package staging

import (
	"encoding/json"
	"fmt"
	"log"
	"mikutool/config"
	"mikutool/public/util"
	"time"
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
	line := 10000
	start := config.Startid
	retryCnt := 0
	for i := start; i < 1000; i++ {
		start := time.Now()
		addr := fmt.Sprintf("http://127.0.0.1:7275/v1/namespaces/%s/devices?line=%d&offset=%d&state=online",
			config.App, line, (i-1)*line)
		log.Println("Request URL:", addr)
		config.Body = ""
		resp, err := util.HttpReq("GET", addr, "", config.HeaderMap)
		if err != nil {
			log.Println("http request failed:", err)
			return
		}
		log.Println("Request took:", time.Since(start))

		var devices Devices
		if err := json.Unmarshal([]byte(resp), &devices); err != nil {
			log.Println("json unmarshal failed:", err)
			return
		}

		log.Println("Response:", len(devices.Items))
		if len(devices.Items) != line {
			i = i - 1
			time.Sleep(time.Duration(retryCnt) * 10 * time.Second)
			retryCnt++
			log.Println("Retry count:", retryCnt)
			continue
		}
		for _, device := range devices.Items {
			if device.AlarmTypesForSnap != "" {
				fmt.Println(device)
				//config.Method = "PATCH"
				addr := fmt.Sprintf("http://127.0.0.1:7275/v1/namespaces/%s/devices/%s",
					config.App, device.Gbid)
				body := `{"operations":[{
							"key":"alarmTypesForSnap",
							"op":"replace",
							"value":"" 
						}]}`
				headerMap := config.HeaderMap
				//headerMap["Content-Type"] = "application/json"
				_, err := util.HttpReq("PATCH", addr, body, headerMap)
				if err != nil {
					log.Println("http request failed:", err)
					return
				}
				processed++
				log.Println("Processed:", processed)
				time.Sleep(30 * time.Millisecond)
				//return
				//if processed >= 10 {
				//	return
				//}
			}
		}
		log.Println("Processing batch:", i, devices.Total/line, devices.Total)
		//log.Println("total:", devices.Total)
		if (i-1)*line >= devices.Total {
			break
		}
		time.Sleep(time.Second * 3)
	}
}

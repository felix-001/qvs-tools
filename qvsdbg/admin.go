package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type Device struct {
	NodeId string `json:"nodeId"`
}

func getDevice(gbid string) (*Device, error) {
	cmd := fmt.Sprintf("curl -s --location --request GET http://10.20.76.42:7277/v1/devices/%s --header 'authorization: QiniuStub uid=1'", gbid)
	result, err := jumpboxCmd(cmd)
	if err != nil {
		log.Println(result, err, cmd)
		return nil, err
	}
	//log.Println("result:", result)
	device := &Device{}
	if err := json.Unmarshal([]byte(result), device); err != nil {
		return nil, err
	}
	return device, err
}

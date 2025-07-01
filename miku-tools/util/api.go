package util

import (
	"fmt"
)

func GetDomain() {
	addr := fmt.Sprintf("http://%mls.cn-east-1.qiniumiku.com/?domainConfig&name=%s", Conf.Bucket, Conf.Domain)
	resp, err := S3get(addr, Conf)
	if err != nil {
		return
	}
	fmt.Println(resp)
}

func UpdateDomain() {
	addr := fmt.Sprintf("http://%mls.cn-east-1.qiniumiku.com/?domainConfig&name=%s", Conf.Bucket, Conf.Domain)
	body := `{"streamConf": {"enableXsStream": true}}`
	resp, err := S3patch(addr, body, Conf)
	if err != nil {
		return
	}
	fmt.Println(resp)
}

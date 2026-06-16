package miku

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

func (m *Miku) DnsLog() {
	if m.conf.Domain == "" {
		log.Println("domain is empty")
		return
	}
	credential := common.NewCredential(m.conf.DnsPod.SecretID, m.conf.DnsPod.SecretKey)
	client, err := dnspod.NewClient(credential, "", profile.NewClientProfile())
	if err != nil {
		log.Println("create dnspod client err:", err)
		return
	}

	const pageSize = 500
	domain := common.StringPtr(m.conf.Domain)

	firstReq := dnspod.NewDescribeDomainLogListRequest()
	firstReq.Domain = domain
	firstReq.Offset = common.Uint64Ptr(0)
	firstReq.Limit = common.Uint64Ptr(1)
	firstResp, err := client.DescribeDomainLogList(firstReq)
	if err != nil {
		log.Println(err)
		return
	}
	if firstResp.Response == nil || firstResp.Response.TotalCount == nil {
		log.Println("TotalCount is empty")
		return
	}
	totalCount := *firstResp.Response.TotalCount
	log.Printf("total dns log count: %d\n", totalCount)

	var allLogs []string
	for offset := uint64(0); offset < totalCount; offset += pageSize {
		if offset > 0 {
			time.Sleep(20 * time.Millisecond)
		}
		req := dnspod.NewDescribeDomainLogListRequest()
		req.Domain = domain
		req.Offset = common.Uint64Ptr(offset)
		req.Limit = common.Uint64Ptr(pageSize)
		resp, err := client.DescribeDomainLogList(req)
		if err != nil {
			log.Println(err)
			return
		}
		if resp.Response == nil || len(resp.Response.LogList) == 0 {
			break
		}
		for _, item := range resp.Response.LogList {
			if item != nil {
				allLogs = append(allLogs, *item)
			}
		}
	}

	result := map[string]interface{}{
		"TotalCount": totalCount,
		"LogList":    allLogs,
	}
	bytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

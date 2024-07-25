package main

import (
	"encoding/json"
	"fmt"
)

// bucket - stream - node

func (s *NetprobeSrv) StreamReport(paramMap map[string]string) string {
	body := paramMap["body"]
	stream := paramMap["stream"]
	bucket := paramMap["bucket"]

	if _, ok := s.streamReportMap[bucket]; !ok {
		s.streamReportMap[bucket] = make(map[string]map[string]map[string]int)
	}
	if _, ok := s.streamReportMap[bucket][stream]; !ok {
		s.streamReportMap[bucket][stream] = make(map[string]map[string]int)
	}

	nodeStreamInfoMap := make(map[string]map[string]int)

	//ipOnlineNumMap := map[string]int{}
	if err := json.Unmarshal([]byte(body), &nodeStreamInfoMap); err != nil {
		return fmt.Sprintf("unmashal err, %v", err)
	}
	for nodeId, ips := range nodeStreamInfoMap {
		s.streamReportMap[bucket][stream][nodeId] = ips
	}
	return "success"
}

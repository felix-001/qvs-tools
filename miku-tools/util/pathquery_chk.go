package util

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"

	"github.com/qbox/mikud-live/common/model"
)

func pathqueryChk() {
	lines := LoadElkLog(Conf.PathqueryLogFile)
	log.Println("total line count:", len(lines))
	for _, line := range lines[1:] {
		//fmt.Println(line)
		raw := getFullPathJson(line)
		//fmt.Println(raw)
		var resp model.PathQueryResponse
		if err := json.Unmarshal([]byte(raw), &resp); err != nil {
			log.Println(err)
			return
		}
		//log.Println(*resp.Sources[0])
		fullPathChk(resp)
	}

}

func fullPathChk(resp model.PathQueryResponse) {
	for _, source := range resp.Sources[1:] {
		u, err := url.Parse(source.Url)
		if err != nil {
			log.Println(err, source.Url)
			return
		}
		isp, area, _ := GetLocate(u.Hostname(), IpParser)
		localIsp, localArea, _ := GetLocate(source.BindLocalIp, IpParser)
		if isp != localIsp {

		}
		if area != localArea &&
			source.Node != "16234ef0-03f7-38f6-9bd7-003d0ba2081e-vdn-jsyz1-dls-1-9" &&
			source.Node != "886fff44-603b-393c-a169-a49d24bcbf0c-vdn-jsyz1-dls-1-8" {
			log.Println("area not match", area, localArea, u.Hostname(), source.BindLocalIp, source.Node, source, resp.ConnectId)
		}

	}
}

func getFullPathJson(input string) string {
	start := strings.Index(input, "resp:")
	if start == -1 {
		log.Println("can't found \"resp:\"")
		return ""
	}
	start += len("resp:")
	raw := input[start:]
	end := strings.Index(raw, "connectId=")
	if end == -1 {
		log.Println("can't found connectId")
		return ""
	}
	raw = raw[:end-1]
	return raw
}

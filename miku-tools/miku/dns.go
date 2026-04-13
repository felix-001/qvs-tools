package miku

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/qbox/pili/base/qiniu/xlog.v1"
)

func (m *Miku) DumpDns() {
	if m.conf.Domain == "" {
		log.Println("domain is empty")
		return
	}
	xl := xlog.NewDummyWithCtx(context.Background())
	resp, err := m.resources.DnsPodCli.GetRecords(xl, m.conf.Domain, m.conf.Host, "", 0)
	if err != nil {
		log.Println(err)
		return
	}
	bytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(bytes))
}

package main

import (
	"context"
	"log"
	"strconv"

	"github.com/qbox/bo-sdk/base/xlog.v1"
	"github.com/qbox/bo-sdk/sdk/qconf/appg"
	"github.com/qbox/bo-sdk/sdk/qconf/qconfapi"
)

func (s *Parser) GetAkSk() {
	qc := qconfapi.New(&s.conf.AccountCfg)
	ag := appg.Client{Conn: qc}
	uid, err := strconv.Atoi(s.conf.Uid)
	if err != nil {
		log.Fatalln(err)
	}
	ak, sk, err := ag.GetAkSk(xlog.FromContextSafe(context.Background()), uint32(uid))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("ak:", ak, "sk:", sk)
}

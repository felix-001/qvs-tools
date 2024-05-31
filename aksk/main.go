package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/qbox/pili/base/qbox.us/api/qconf/appg"
	"github.com/qbox/pili/base/qbox.us/qconf/qconfapi"
	"github.com/qbox/pili/base/qiniu/xlog.v1"
	qconfig "github.com/qiniu/x/config"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("args: <uid>")
		return
	}
	var cfg qconfapi.Config
	err := qconfig.LoadFile(&cfg, "/usr/local/miku-admin.conf")
	if err != nil {
		log.Fatalf("load config file failed: %s\n", err.Error())
	}
	qc := qconfapi.New(&cfg)
	ag := appg.Client{Conn: qc}
	uid, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	ak, sk, err := ag.GetAkSk(xlog.FromContextSafe(context.Background()), uint32(uid))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("ak:", ak, "sk:", sk)
}

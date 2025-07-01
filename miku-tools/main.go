package main

import (
	"log"
	"mikutool/manager"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmdMgr := manager.NewCommandManager()
	cmdMgr.Init()
	cmdMgr.Register()
	cmdMgr.Exec()
}

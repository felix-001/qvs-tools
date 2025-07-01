package main

import (
	"mikutool/manager"
)

func main() {
	cmdMgr := manager.NewCommandManager()
	cmdMgr.Init()
	cmdMgr.Register()
	cmdMgr.Exec()
}

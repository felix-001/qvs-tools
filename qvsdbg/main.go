package main

import (
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := Config{}
	if err := loadConf(&conf); err != nil {
		log.Println(err)
		return
	}

	parser := NewParser(&conf)
	parser.Run()
}

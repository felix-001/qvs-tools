package miku

import "mikutool/config"

func Publish(conf *config.Config) {
	Build(conf)
}

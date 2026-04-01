package miku

import "mikutool/config"

func Publish(conf *config.Config) {
	CreateIssue("miku", "发布", "发布", "task")
}

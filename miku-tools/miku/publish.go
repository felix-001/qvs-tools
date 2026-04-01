package miku

import "mikutool/config"

func Publish(conf *config.Config) {
	CreateIssue("MIKU", "发布1", "发布1", "task")
	GetProjects()
	GetIssue()
}

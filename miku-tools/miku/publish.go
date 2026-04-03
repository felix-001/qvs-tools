package miku

import (
	"log"
	"mikutool/config"
)

func Publish(conf *config.Config) {
	if conf.Pr == 0 {
		log.Println("need -pr")
		return
	}
	githubMgr := NewGitHubMgr(conf.GitHubConf)
	prTitle := githubMgr.GetPrTitle(conf.Pr)
	log.Println(prTitle)
	ResetGitLab()
	OpenConfEdit(conf)
}

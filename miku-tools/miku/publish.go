package miku

import (
	"log"
	"mikutool/config"
)

func Publish(conf *config.Config) {
	//Build(conf)
	/*
		packageName, err := GetOldPackageName(conf)
		if err != nil {
			log.Printf("获取旧包名失败: %v\n", err)
			return
		}
		log.Printf("旧包名: %s\n", packageName)
	*/
	//UpdatePackageName(conf, "MIKUD_LIVE.2026-04-02-16-11-20.tar.gz")
	githubMgr := NewGitHubMgr(conf.GitHubConf)
	prTitle := githubMgr.GetPrTitle(3122)
	log.Println(prTitle)
}

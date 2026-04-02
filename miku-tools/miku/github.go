package miku

import (
	"context"
	"fmt"
	"mikutool/config"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type GitMergeRequest struct {
	GiraId        string
	GiraUrl       string //https://jira.qiniu.io/browse/-277
	Pr            *github.PullRequest
	GitId         string
	Title         string
	Author        string
	CreateAt      *time.Time
	MergeMessage  string // 本次合并的提交信息, 变更内容
	ChangeModules string // 变更模块
}

type GitHubMgr struct {
	Client *github.Client
	Conf   config.GitHubConf
}

func NewGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	if client == nil {
		log.Error().Msg("创建 GitHub 客户端失败")
		return nil
	}
	return client
}

func NewGitHubMgr(conf config.GitHubConf) *GitHubMgr {
	client := NewGitHubClient(context.Background(), conf.GitHubToken)
	if client == nil {
		log.Error().Msg("创建 GitHub 客户端失败")
		return nil
	}
	return &GitHubMgr{
		Client: client,
		Conf:   conf}
}

func (s *GitHubMgr) GetGitHubPullRequest() map[string]*GitMergeRequest {
	// 获取仓库信息
	repository, _, err := s.Client.Repositories.Get(context.Background(), s.Conf.Owner, s.Conf.Repo)
	if err != nil {
		log.Error().Err(err).Msg("获取仓库信息失败")
	}
	log.Info().Str("url", *repository.URL).Msg("仓库 URL")

	// 获取最近的 PR 记录
	opt := &github.PullRequestListOptions{
		State: "all", // 可选值：open, closed, all
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 30, // 根据需要调整每页显示的数量
		},
	}

	prs, resp, err := s.Client.PullRequests.List(context.Background(), s.Conf.Owner, s.Conf.Repo, opt)
	if err != nil {
		log.Error().Err(err).Msg("获取 PR 列表失败")
	}
	defer resp.Body.Close()

	// 打印 PR 信息
	mrMap := make(map[string]*GitMergeRequest)
	for _, pr := range prs {
		IsMerged := pr.MergedAt != nil && pr.GetState() == "closed"
		if !IsMerged {
			continue
		}
		// 是否以开头
		if strings.Index(*pr.Title, "-") != 0 {
			continue
		}

		giraId, changeModules, err := ParseTitle(*pr.Title)
		if err != nil {
			log.Warn().Err(err).Str("title", *pr.Title).Msg("解析 PR 标题失败")
		}

		giraurl := ""
		if giraId != "" && giraId != "0" {
			giraurl = fmt.Sprintf("https://jira.qiniu.io/browse/-%s", giraId)
		}

		mergeMessage := ""
		if pr.Body != nil {
			mergeMessage = *pr.Body
		}

		newPr := &GitMergeRequest{
			GiraId:        giraId,
			GiraUrl:       giraurl,
			Pr:            pr,
			GitId:         fmt.Sprint(*pr.Number),
			Title:         *pr.Title,
			Author:        *pr.GetUser().Login,
			CreateAt:      pr.CreatedAt,
			MergeMessage:  mergeMessage,
			ChangeModules: changeModules,
		}

		/*
			log.Logger.Info().
				Str("GiraId", newPr.GiraId).
				Str("GiraUrl", newPr.GiraUrl).
				Str("Title", newPr.Title).
				Str("Author", newPr.Author).
				Str("Merge", TimeToBeijing(*newPr.Pr.MergedAt)).
				Str("CreateAt", TimeToBeijing(*newPr.CreateAt)).
				Str("MergeMessage", newPr.MergeMessage).
				Str("ChangeModules", newPr.ChangeModules).
				Msgf("GitHub Pull Request\n")
		*/
		mrMap[fmt.Sprint(*pr.Number)] = newPr
	}
	return mrMap
}

func ParseTitle(mrTitle string) (GiraId string, changeModules string, err error) {
	GiraId = ParseGiraIdFromTitle(mrTitle)

	// 定义正则表达式，匹配中括号内的内容
	re := regexp.MustCompile(`\[(.*?)\]`)

	// 查找所有匹配的内容
	matches := re.FindAllStringSubmatch(mrTitle, -1)
	if len(matches) > 0 {
		changeModules = matches[0][1]
	}
	return GiraId, changeModules, nil
}

func ParseGiraIdFromTitle(mrTitle string) string {
	if !strings.HasPrefix(mrTitle, "MIKU-") {
		return ""
	}
	parts := strings.Split(mrTitle, " ")
	if len(parts) < 2 {
		return ""
	}
	matchkey := parts[0][5:]
	return matchkey
}

func (s *GitHubMgr) GetPrTitle(num int) string {
	pr, _, err := s.Client.PullRequests.Get(context.Background(), s.Conf.Owner, s.Conf.Repo, num)
	if err != nil {
		log.Error().Err(err).Msg("获取 PR 详情失败")
	}
	return *pr.Title
}

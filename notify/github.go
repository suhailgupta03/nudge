package notify

import (
	"log"
	"nudge/internal/database/pr"
	"nudge/internal/database/repository"
	provider "nudge/internal/provider/github"

	"github.com/knadh/koanf/v2"
)

type GitHubNotification struct {
	ko *koanf.Koanf
	lo *log.Logger
}

func GithubNotificationInit(ko *koanf.Koanf, lo *log.Logger) *GitHubNotification {
	return &GitHubNotification{
		ko: ko,
		lo: lo,
	}
}

func (n *GitHubNotification) Post(repo repository.RepoModel, pr pr.PRModel, actorToNotify string, isReviewer bool) error {
	jwt, _ := provider.GenerateAppJWT(n.ko.String("app.private_key"), n.ko.String("github.app_id"))
	g := provider.Init(*jwt)
	iToken, appTokenErr := g.GetAppInstallationAccessToken(repo.InstallationId)
	if appTokenErr != nil {
		return appTokenErr
	}

	g = provider.Init(*iToken.Token)
	message := createNotificationMessage(actorToNotify, isReviewer)
	err := g.PostComment(repo.Name, repo.Owner, pr.Number, message)
	return err
}

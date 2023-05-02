package notify

import (
	"log"
	"nudge/internal/database/repository"
	provider "nudge/internal/provider/github"

	"github.com/knadh/koanf/v2"
)

type Notification struct {
	ko *koanf.Koanf
	lo *log.Logger
}

func Init(ko *koanf.Koanf, lo *log.Logger) *Notification {
	return &Notification{
		ko: ko,
		lo: lo,
	}
}

func (n *Notification) Post(repo repository.RepoModel, prNumber int, actorToNotify string, isReviewer bool) error {
	jwt, _ := provider.GenerateAppJWT(n.ko.String("app.private_key"), n.ko.String("github.app_id"))
	g := provider.Init(*jwt)
	iToken, appTokenErr := g.GetAppInstallationAccessToken(repo.InstallationId)
	if appTokenErr != nil {
		return appTokenErr
	}

	g = provider.Init(*iToken.Token)
	message := createNotificationMessage(actorToNotify, isReviewer)
	err := g.PostComment(repo.Name, repo.Owner, prNumber, message)
	return err
}

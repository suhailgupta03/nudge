package notify

import (
	"fmt"
	"nudge/internal/database/repository"
)

type Notify interface {
	Post(repo repository.RepoModel, prNumber int, actorToNotify string, isReviewer bool) error
}

func createNotificationMessage(actor string, isReviewer bool) string {
	if isReviewer {
		return fmt.Sprintf("Hello @%s. The PR is blocked on your approval. Please review it ASAP.", actor)
	} else {
		return fmt.Sprintf("Hello @%s. The PR is blocked on your changes. Please complete it ASAP.", actor)
	}
}

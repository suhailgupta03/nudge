package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/knadh/koanf/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"log"
	"net/http"
	"nudge/internal/database/pr"
	"nudge/internal/database/repository"
	"nudge/internal/database/user"
	"strconv"
)

type SlackNotification struct {
	ko *koanf.Koanf
	lo *log.Logger
	db *mongo.Database
}

func SlackNotificationInit(ko *koanf.Koanf, lo *log.Logger, db *mongo.Database) *SlackNotification {
	return &SlackNotification{
		ko: ko,
		lo: lo,
		db: db,
	}
}

// Post https://api.slack.com/methods/chat.postMessage
func (s *SlackNotification) Post(repo repository.RepoModel, pr pr.PRModel, actorToNotify string, isReviewer bool) error {
	prLink := fmt.Sprintf("https://github.com/%s/%s/pull/%d", repo.Owner, repo.Name, pr.Number)
	message := createSlackNotificationMessage(actorToNotify, repo.Name, prLink, pr.Number, isReviewer)

	// Fetch slack user details
	userDetails, uErr := user.Init(s.db).FindUserByGitHubUsername(actorToNotify)
	if uErr != nil {
		return uErr
	}

	if userDetails.SlackUserId != nil && userDetails.SlackAccessToken != nil { // This means that
		// the Slack app has been installed
		channel := *userDetails.SlackUserId
		if userDetails.GitHubUsername != actorToNotify && userDetails.GithubSlackMapping != nil {
			// If the actor to notify is not the root user, extract the slack user id
			// from the stored mapping
			for _, m := range *userDetails.GithubSlackMapping {
				if m.GitHubUsername == actorToNotify {
					channel = m.SlackUserId
					break
				}
			}
		}
		postBody, _ := json.Marshal(map[string]string{
			"text":    message,
			"channel": channel,
		})

		bearer := fmt.Sprintf("Bearer %s", *userDetails.SlackAccessToken)
		req, _ := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(postBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", bearer)
		client := &http.Client{}
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return errors.New("Failed with status code as " + strconv.Itoa(resp.StatusCode))
		}

		_, rErr := io.ReadAll(resp.Body)
		return rErr
	} else {
		return nil
	}

}

func createSlackNotificationMessage(actor, repoName, prLink string, prNumber int, isReviewer bool) string {
	actionVerb := "changes"
	if isReviewer {
		actionVerb = "approval"
	}

	return fmt.Sprintf("Hello %s. PR <%s|#%d> in repository *%s* is blocked on your %s. Please review it ASAP.", actor, prLink, prNumber, repoName, actionVerb)
}

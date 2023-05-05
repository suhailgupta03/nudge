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
	message := createNotificationMessage(actorToNotify, isReviewer)
	prLink := fmt.Sprintf("https://github.com/%s/%s/pull/%d", repo.Owner, repo.Name, pr.Number)
	message += fmt.Sprintf(" %s", prLink)

	// Fetch slack user details
	userDetails, uErr := user.Init(s.db).FindUserByGitHubUsername(actorToNotify)
	if uErr != nil {
		return uErr
	}

	if userDetails.SlackUserId != nil && userDetails.SlackAccessToken != nil {
		channel := *userDetails.SlackUserId
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

package notify

import (
	"fmt"
	"nudge/internal/database/pr"
	"nudge/internal/database/repository"
	"nudge/internal/database/user"
	"time"
)

type Notify interface {
	Post(repo repository.RepoModel, pr pr.PRModel, actorToNotify string, isReviewer bool) error
}

type NotificationHours interface {
	IsWithinBusinessHours(userTimezone string, businessHours user.NotificationBusinessHours, currentTime time.Time) (bool, error)
}

func createNotificationMessage(actor string, isReviewer bool) string {
	if isReviewer {
		return fmt.Sprintf("Hello @%s. The PR is blocked on your approval. Please review it ASAP.", actor)
	} else {
		return fmt.Sprintf("Hello @%s. The PR is blocked on your changes. Please complete it ASAP.", actor)
	}
}

type BusinessHours struct{}

// IsWithinBusinessHours checks if the currentTime lies within the business hours for the timezone passed
// Note: Unless needed, currentTime must always reflect the current moment in time
func (n *BusinessHours) IsWithinBusinessHours(userTimezone string, businessHours user.NotificationBusinessHours, currentTime time.Time) (bool, error) {
	within := false

	now := currentTime
	location, locationErr := time.LoadLocation(userTimezone)
	if locationErr != nil {
		// Looks like the timezone passed is incorrect
		return within, locationErr
	}

	transportedNow := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Nanosecond(),
		location)

	if businessHours.StartHours <= transportedNow.Hour() && businessHours.EndHours >= transportedNow.Hour() {
		// This means that the current time falls within the business hours
		within = true
	}

	return within, nil
}

package notify

import (
	"fmt"
	"log"
	"nudge/internal/database/pr"
	"nudge/internal/database/repository"
	"nudge/internal/database/user"
	"strings"
	"time"
)

type Notify interface {
	Post(repo repository.RepoModel, pr pr.PRModel, actorToNotify string, isReviewer bool) error
}

func createNotificationMessage(actor string, isReviewer bool) string {
	if isReviewer {
		return fmt.Sprintf("Hello @%s. The PR is blocked on your approval. Please review it ASAP.", actor)
	} else {
		return fmt.Sprintf("Hello @%s. The PR is blocked on your changes. Please complete it ASAP.", actor)
	}
}

func createNotificationMessageWithMultipleActors(actors []string, isReviewer bool) string {
	if isReviewer {
		actorStr := ""
		for _, actor := range actors {
			actorStr += fmt.Sprintf("@%s ", actor)
		}
		actorStr = strings.TrimSpace(actorStr)
		return fmt.Sprintf("Hello %s. The PR is blocked on your approval. Please review it ASAP.", actorStr)
	} else {
		actorStr := actors[0]
		return fmt.Sprintf("Hello @%s. The PR is blocked on your changes. Please complete it ASAP.", actorStr)
	}
}

type NotificationHours interface {
	IsWithinBusinessHours(userTimezone string, businessHours user.NotificationBusinessHours, currentTime time.Time) (bool, error)
}

type BusinessHours struct{}

// IsWithinBusinessHours checks if the currentTime lies within the business hours for the timezone passed
// Note: Unless needed, currentTime must always reflect the current moment in time
func (n *BusinessHours) IsWithinBusinessHours(userTimezone string, businessHours user.NotificationBusinessHours, currentTime time.Time) (bool, error) {
	within := false

	location, locationErr := time.LoadLocation(userTimezone)
	if locationErr != nil {
		// Looks like the timezone passed is incorrect
		return within, locationErr
	}

	transportedNow := currentTime.In(location)

	if businessHours.StartHours <= transportedNow.Hour() && businessHours.EndHours >= transportedNow.Hour() {
		// This means that the current time falls within the business hours
		within = true
	}

	return within, nil
}

type NotificationDaysService interface {
	IsSunday(zone *user.TimeZone, currentTime time.Time) bool
	IsAnyDayInList(zone *user.TimeZone, currentTime time.Time, days []int) bool
}
type NotificationDays struct {
	Lo *log.Logger
}

func (nd *NotificationDays) IsSunday(zone *user.TimeZone, currentTime time.Time) bool {
	loc, err := time.LoadLocation(strings.TrimSpace(string(*zone)))
	if err != nil {
		nd.Lo.Printf("Incorrect timezone passed. Unable to determine if it is Sunday. - %v", err)
		// Will return false, if not able to determine
		return false
	} else {
		currentTimeLocation := currentTime.In(loc)
		if currentTimeLocation.Weekday() == time.Sunday {
			return true
		} else {
			return false
		}
	}
}

func (nd *NotificationDays) IsAnyDayInList(zone *user.TimeZone, currentTime time.Time, days []int) bool {
	loc, err := time.LoadLocation(strings.TrimSpace(string(*zone)))
	if err != nil {
		nd.Lo.Printf("Incorrect timezone passed. Unable to determine the day - %v", err)
		// Will return false, if not able to determine
		return false
	} else {
		currentTimeLocation := currentTime.In(loc)
		for _, day := range days {
			if currentTimeLocation.Weekday() == time.Weekday(day) {
				return true
			}
		}
		return false
	}
}

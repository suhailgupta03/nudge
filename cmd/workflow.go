package main

import (
	"nudge/activity"
	"nudge/actor"
	prm "nudge/internal/database/pr"
	"nudge/internal/database/repository"
	"nudge/internal/database/user"
	time2 "nudge/internal/time"
	"nudge/notify"
	"time"
)

type WorkflowDependencies struct {
	Activity          *activity.Activity
	ActorIdentifier   actor.ActorIdentifier
	NotificationHours notify.NotificationHours
	NotificationDays  notify.NotificationDaysService
	User              *user.User
}

func Workflow(workflowDependencies WorkflowDependencies) {

	start := time.Now().Unix()
	// 1. Determine lifetime effort

	// 2. Check for activity
	delayedPRs, actErr := workflowDependencies.Activity.ActivityCheckTrigger()
	lo.Printf("Found %d delayed PRs", len(*delayedPRs))
	if actErr != nil {
		lo.Printf("Error while trying to run the activity trigger %v", actErr)
		return
	}

	// 3. Identify the actors to notify
	for _, pr := range *delayedPRs {
		lo.Printf("Starting for PR#%d in repository %s", pr.DelayedPR.Number, pr.Repository.Name)
		actorDetails, ierr := workflowDependencies.ActorIdentifier.IdentifyActors(pr.DelayedPR, pr.Repository, ko)
		if ierr != nil {
			lo.Printf("Failed to identify actors for PR %d and repo %s", pr.DelayedPR.Number, pr.Repository.Name)
			continue
		}
		if len(actorDetails) > 0 {
			tz, bizHours := getUserTimezoneDetails(pr.Repository.InstallationId, workflowDependencies.User)
			if len(ko.Ints("bot.skip_days")) > 0 {
				if workflowDependencies.NotificationDays.IsAnyDayInList(tz, time.Now(), ko.Ints("bot.skip_days")) {
					// Do not send a nudge on the days mentioned in the configuration
					lo.Printf("Skipping PR#%d of %s on the days mentioned in the configuration", pr.DelayedPR.Number, pr.Repository.Name)
					continue
				}
			}

			if pr.DelayedPR.TotalBotComments != nil {
				if *pr.DelayedPR.TotalBotComments >= ko.Int("bot.follow_up_threshold_comments") {
					// Since this has exceeded the total number of comments a bot
					// can make, will no longer be sending the nudges
					lo.Printf("Skipping PR#%d of %s since it crossed the threshold", pr.DelayedPR.Number, pr.Repository.Name)
					continue
				}
			}

			if pr.DelayedPR.LastBotCommentMadeAt != nil {
				nt := new(time2.NudgeTime)
				elapsedHoursSinceLastComment := float64((nt.Now().Unix() - *pr.DelayedPR.LastBotCommentMadeAt) / 3600)
				if elapsedHoursSinceLastComment < ko.Float64("bot.interval_to_wait.time") {
					// Do not send a nudge,
					// since the comment made is very recent
					lo.Printf("Skipping PR#%d of %s since the comment made by bot is very recent (%f)hours", pr.DelayedPR.Number, pr.Repository.Name, elapsedHoursSinceLastComment)
					continue
				}
			}

			withinBizHours, _ := workflowDependencies.NotificationHours.IsWithinBusinessHours(string(*tz), *bizHours, time.Now())
			if !withinBizHours {
				// Skip the nudge if not within business hours
				lo.Printf("Skipping PR#%d of %s since outside business hours (%d-%d) %s", pr.DelayedPR.Number, pr.Repository.Name, (*bizHours).StartHours, (*bizHours).EndHours, string(*tz))
				continue
			}

			actor := actorDetails[0].GithubUserName
			isReviewer := actorDetails[0].IsReviewer
			lo.Printf("Review is stuck because of %s", actor)
			// 4. Notify the actors blocking the PR
			postNotifications(pr.Repository, pr.DelayedPR, actor, isReviewer)
			/**
			After the notifications have been sent:
			- Increment the comment counter for this PR
			- Updates the total comments made
			*/
			updateCommentMeta(pr.DelayedPR)
		}
	}

	lo.Printf("Completed the workflow in %v seconds", time.Now().Unix()-start)
}

// postNotifications sends notifications on GitHub and Slack (if activated). This is the last step in the workflow
func postNotifications(repository repository.RepoModel, delayedPR prm.PRModel, actor actor.GithubUserName, isReviewer bool) {
	n := notify.GithubNotificationInit(ko, lo)
	postErr := n.Post(repository, delayedPR, string(actor), isReviewer)
	if postErr != nil {
		lo.Printf("Failed to post a message to the actor blocking the PR %v", postErr)
	}

	s := notify.SlackNotificationInit(ko, lo, database)
	slackErr := s.Post(repository, delayedPR, string(actor), isReviewer)
	if slackErr != nil {
		lo.Printf("Failed to post a message to slack %v", slackErr)
	}
}

func updateCommentMeta(pr prm.PRModel) {
	prm.Init(database).IncrementTotalCommentsMade(pr.PRID)
}

// getUserTimezoneDetails returns the timezone and business hours stored for the user. If the timezone
// is invalid or notification hours are not stored, the default as defined in the config are returned.
func getUserTimezoneDetails(installationId int64, service user.UserTimezoneService) (*user.TimeZone, *user.NotificationBusinessHours) {
	tz, bizHours, err := service.FindUserTimezoneByInstallationId(installationId)
	if err != nil {
		lo.Printf("Failed to find the user timezone and business hours. Using default. - %v", err)
		return getDefaultTimezoneDetails()
	} else {
		if tz != nil && bizHours != nil {
			_, lErr := time.LoadLocation(string(*tz))
			if lErr != nil || bizHours.EndHours == 0 || bizHours.StartHours == 0 {
				lo.Printf("Error loading %s. Using default. - %v", string(*tz), lErr)
				return getDefaultTimezoneDetails()
			}
			return tz, bizHours
		} else {
			lo.Printf("Using default timezone details for installation %d", installationId)
			return getDefaultTimezoneDetails()
		}
	}
}

func getDefaultTimezoneDetails() (*user.TimeZone, *user.NotificationBusinessHours) {
	tz := user.TimeZone(ko.String("bot.default_timezone"))
	bh := user.NotificationBusinessHours{
		StartHours: ko.Int("bot.default_business_hours.start"),
		EndHours:   ko.Int("bot.default_business_hours.end"),
	}
	return &tz, &bh
}

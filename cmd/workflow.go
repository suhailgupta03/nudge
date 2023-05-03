package main

import (
	"nudge/activity"
	"nudge/actor"
	"nudge/notify"
	"time"
)

func Workflow() {

	start := time.Now().Unix()
	// 1. Determine lifetime effort

	// 2. Check for activity
	act := activity.Init(ko, database, lo)
	delayedPRs, actErr := act.ActivityCheckTrigger()
	if actErr != nil {
		lo.Printf("Error while trying to run the activity trigger %v", actErr)
		return
	}

	// 3. Identify the actors to notify
	for _, pr := range *delayedPRs {
		actorDetails, ierr := actor.IdentifyActors(pr.DelayedPR, pr.Repository, ko)
		if ierr != nil {
			lo.Printf("Failed to identify actors for PR %d and repo %s", pr.DelayedPR.Number, pr.Repository.Name)
			continue
		}
		if len(actorDetails) > 0 {
			actor := actorDetails[0].GithubUserName
			isReviewer := actorDetails[0].IsReviewer
			lo.Printf("Review is stuck because of %s", actor)
			n := notify.GithubNotificationInit(ko, lo)
			// 4. Notify the actors blocking the PR

			postErr := n.Post(pr.Repository, pr.DelayedPR, string(actor), isReviewer)
			if postErr != nil {
				lo.Printf("Failed to post a message to the actor blocking the PR %v", postErr)
			}

			s := notify.SlackNotificationInit(ko, lo, database)
			slackErr := s.Post(pr.Repository, pr.DelayedPR, string(actor), isReviewer)
			if slackErr != nil {
				lo.Printf("Failed to post a message to slack %v", slackErr)
			}

		}
	}

	lo.Printf("Completed the workflow in %v seconds", time.Now().Unix()-start)
}

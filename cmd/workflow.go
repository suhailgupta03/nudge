package main

import (
	"nudge/activity"
	"nudge/actor"
)

func Workflow() {

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
		actorDetails, _ := actor.IdentifyActors(pr.DelayedPR, pr.Repository, ko)
		lo.Printf("Review is stuck because of %s", actorDetails)
	}
}

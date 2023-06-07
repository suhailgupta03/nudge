package time

import (
	"log"
	"time"
)

type NudgeTime struct{}

type NudgeTimeService interface {
	NudgeTime() time.Time
}

func (nt *NudgeTime) NudgeTime() *time.Time {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		log.Printf("Failed to load initialize nudge time %v", err)
		return nil
	}
	log.Printf("Loaded local timezone (from nudge-time) %s", loc.String())
	t := time.Now().In(loc)
	return &t
}

func (nt *NudgeTime) Now() *time.Time {
	return nt.NudgeTime()
}

package notify

import (
	"errors"
	"log"
	"nudge/internal/database/user"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateNotificationMessage(t *testing.T) {
	testCases := []struct {
		actor      string
		isReviewer bool
		expected   string
	}{
		{
			actor:      "John",
			isReviewer: true,
			expected:   "Hello @John. The PR is blocked on your approval. Please review it ASAP.",
		},
		{
			actor:      "Jane",
			isReviewer: false,
			expected:   "Hello @Jane. The PR is blocked on your changes. Please complete it ASAP.",
		},
	}

	for _, tc := range testCases {
		actual := createNotificationMessage(tc.actor, tc.isReviewer)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestIsWithinBusinessHours(t *testing.T) {
	testCases := []struct {
		description    string
		userTimezone   string
		businessHours  user.NotificationBusinessHours
		expectedResult bool
		currentDate    time.Time
		expectedError  error
	}{
		{
			description:  "Valid timezone, within business hours",
			userTimezone: "America/New_York",
			businessHours: user.NotificationBusinessHours{
				StartHours: 9,
				EndHours:   17,
			},
			expectedResult: true,
			currentDate:    time.Date(2022, 1, 1, 15, 0, 0, 0, time.UTC),
			expectedError:  nil,
		},
		{
			description:  "Valid timezone, outside business hours (before start time)",
			userTimezone: "America/New_York",
			businessHours: user.NotificationBusinessHours{
				StartHours: 11,
				EndHours:   17,
			},
			expectedResult: false,
			currentDate:    time.Date(2022, 1, 1, 10, 0, 0, 0, time.UTC),
			expectedError:  nil,
		},
		{
			description:  "Valid timezone, outside business hours (after end time)",
			userTimezone: "America/New_York",
			businessHours: user.NotificationBusinessHours{
				StartHours: 9,
				EndHours:   17,
			},
			expectedResult: false,
			currentDate:    time.Date(2022, 1, 1, 23, 0, 0, 0, time.UTC),
			expectedError:  nil,
		},
		{
			description:  "Valid timezone, same start and end times for business hours",
			userTimezone: "America/New_York",
			businessHours: user.NotificationBusinessHours{
				StartHours: 9,
				EndHours:   9,
			},
			expectedResult: false,
			currentDate:    time.Date(2022, 1, 1, 10, 0, 0, 0, time.UTC),
			expectedError:  nil,
		},
		{
			description:  "Invalid timezone",
			userTimezone: "Invalid/Timezone",
			businessHours: user.NotificationBusinessHours{
				StartHours: 9,
				EndHours:   17,
			},
			expectedResult: false,
			currentDate:    time.Date(2022, 1, 1, 10, 0, 0, 0, time.UTC),
			expectedError:  errors.New("unknown time zone Invalid/Timezone"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			result, err := new(BusinessHours).IsWithinBusinessHours(testCase.userTimezone, testCase.businessHours, testCase.currentDate)

			if testCase.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, testCase.expectedError.Error(), err.Error())
			}

			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}

func TestIsSunday(t *testing.T) {
	testCases := []struct {
		name        string
		timezone    string
		currentTime time.Time
		expected    bool
		errExpected bool
	}{
		{
			name:        "ValidTimezone_Sunday",
			timezone:    "Asia/Kolkata",
			currentTime: time.Date(2022, 11, 6, 0, 0, 0, 0, time.UTC),
			expected:    true,
		},
		{
			name:        "ValidTimezone_NotSunday",
			timezone:    "Asia/Kolkata",
			currentTime: time.Date(2022, 11, 5, 0, 0, 0, 0, time.UTC),
			expected:    false,
		},
		{
			name:        "InvalidTimezone",
			timezone:    "Invalid/Timezone",
			currentTime: time.Date(2022, 11, 6, 0, 0, 0, 0, time.UTC),
			expected:    false,
			errExpected: true,
		},
	}

	notificationDays := &NotificationDays{
		Lo: log.New(os.Stderr, "TEST: ", log.LstdFlags),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			timezone := user.TimeZone(tc.timezone)
			result := notificationDays.IsSunday(&timezone, tc.currentTime)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateNotificationMessageWithMultipleActors(t *testing.T) {
	testCases := []struct {
		name       string
		actors     []string
		isReviewer bool
		expected   string
	}{
		{
			name:       "MultipleActors_IsReviewer",
			actors:     []string{"actor1", "actor2"},
			isReviewer: true,
			expected:   "Hello @actor1 @actor2. The PR is blocked on your approval. Please review it ASAP.",
		},
		{
			name:       "SingleActor_IsReviewer",
			actors:     []string{"actor1"},
			isReviewer: true,
			expected:   "Hello @actor1. The PR is blocked on your approval. Please review it ASAP.",
		},
		{
			name:       "SingleActor_NotReviewer",
			actors:     []string{"actor1"},
			isReviewer: false,
			expected:   "Hello @actor1. The PR is blocked on your changes. Please complete it ASAP.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := createNotificationMessageWithMultipleActors(tc.actors, tc.isReviewer)
			assert.Equal(t, tc.expected, result)
		})
	}
}

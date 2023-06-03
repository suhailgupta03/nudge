package notify

import (
	"errors"
	"nudge/internal/database/user"
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
			currentDate:    time.Date(2022, 1, 1, 10, 0, 0, 0, time.UTC),
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

package notify

import (
	"testing"

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

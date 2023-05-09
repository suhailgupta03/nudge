package notify

import (
	"strconv"
	"strings"
	"testing"
)

func TestCreateSlackNotificationMessage(t *testing.T) {
	tests := []struct {
		name       string
		actor      string
		repoName   string
		prLink     string
		prNumber   int
		isReviewer bool
	}{
		{
			name:       "Test Case Reviewer",
			actor:      "john",
			repoName:   "test-repo",
			prLink:     "https://github.com/owner/test-repo/pull/123",
			prNumber:   123,
			isReviewer: true,
		},
		{
			name:       "Test Case Assignee",
			actor:      "doe",
			repoName:   "example-repo",
			prLink:     "https://github.com/owner/example-repo/pull/456",
			prNumber:   456,
			isReviewer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := createSlackNotificationMessage(tt.actor, tt.repoName, tt.prLink, tt.prNumber, tt.isReviewer)
			if !strings.Contains(message, tt.actor) || !strings.Contains(message, tt.repoName) || !strings.Contains(message, tt.prLink) {
				t.Errorf("Expected message to contain actor, repoName, and prLink")
			}
			prNumInMsg := strings.Contains(message, "|#"+strconv.Itoa(tt.prNumber))
			actionVerb := "approval"
			if !tt.isReviewer {
				actionVerb = "changes"
			}
			actionPresent := strings.Contains(message, actionVerb)
			if !prNumInMsg || !actionPresent {
				t.Errorf("Expected message to contain PR number and action verb")
			}
		})
	}
}

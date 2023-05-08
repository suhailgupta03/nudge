package actor

import (
	"github.com/google/go-github/v52/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	prp "nudge/internal/database/pr"
	provider "nudge/internal/provider/github"
	"testing"
	"time"
)

func ptrString(s string) *string {
	return &s
}
func ptrInt64(v int64) *int64 {
	return &v
}

type GitHubProviderMock struct {
	mock.Mock
}

func (m *GitHubProviderMock) GetAppInstallationAccessToken(installationID int64) (*provider.GithubAppTokenDetails, error) {
	args := m.Called(installationID)
	return args.Get(0).(*provider.GithubAppTokenDetails), args.Error(1)
}

func (m *GitHubProviderMock) GetPrById(number int, owner string, repo string) (*github.PullRequest, error) {
	args := m.Called(number, owner, repo)
	return args.Get(0).(*github.PullRequest), args.Error(1)
}

func (m *GitHubProviderMock) GetBranchProtection(repo string, ref string, owner string) (*github.Protection, error) {
	args := m.Called(repo, ref, owner)
	return args.Get(0).(*github.Protection), args.Error(1)
}

func TestHasPendingActionItemsForAuthor(t *testing.T) {
	t.Run("nil_reviews", func(t *testing.T) {
		hasPending := hasPendingActionItemsForAuthor(nil)
		assert.False(t, hasPending)
	})

	t.Run("empty_reviews", func(t *testing.T) {
		reviews := []prp.Review{}
		hasPending := hasPendingActionItemsForAuthor(&reviews)
		assert.False(t, hasPending)
	}) //TODO: MODIFY THIS CONDITION IN CODE

	t.Run("all_approved", func(t *testing.T) {
		reviews := []prp.Review{
			{ReviewState: ptrString("approved")},
			{ReviewState: ptrString("approved")},
		}
		hasPending := hasPendingActionItemsForAuthor(&reviews)
		assert.False(t, hasPending)
	})

	t.Run("some_not_approved", func(t *testing.T) {
		reviews := []prp.Review{
			{ReviewState: ptrString("changes_requested")},
			{ReviewState: ptrString("approved")},
		}
		hasPending := hasPendingActionItemsForAuthor(&reviews)
		assert.True(t, hasPending)
	})
}

func TestIsPrApproved(t *testing.T) {
	t.Run("min_reviews_required_zero", func(t *testing.T) {
		isApproved, _ := isPrApproved(nil, 0)
		assert.True(t, isApproved)
	})

	t.Run("nil_reviews", func(t *testing.T) {
		isApproved, _ := isPrApproved(nil, 2)
		assert.False(t, isApproved)
	})

	t.Run("enough_approvals_no_changes_requested", func(t *testing.T) {
		now := time.Now().Unix()

		reviews := []prp.Review{
			{ReviewState: ptrString("approved"), SubmittedAt: ptrInt64(now), Reviewer: ptrString("user1")},
			{ReviewState: ptrString("approved"), SubmittedAt: ptrInt64(now + 1), Reviewer: ptrString("user2")},
		}

		isApproved, _ := isPrApproved(&reviews, 2)
		assert.True(t, isApproved)
	})

	t.Run("not_enough_approvals", func(t *testing.T) {
		now := time.Now().Unix()

		reviews := []prp.Review{
			{ReviewState: ptrString("approved"), SubmittedAt: ptrInt64(now), Reviewer: ptrString("user1")},
		}

		isApproved, _ := isPrApproved(&reviews, 2)
		assert.False(t, isApproved)
	})

	t.Run("changes_requested", func(t *testing.T) {
		now := time.Now().Unix()

		reviews := []prp.Review{
			{ReviewState: ptrString("approved"), SubmittedAt: ptrInt64(now), Reviewer: ptrString("user1")},
			{ReviewState: ptrString("changes_requested"), SubmittedAt: ptrInt64(now + 1), Reviewer: ptrString("user2")},
		}

		isApproved, _ := isPrApproved(&reviews, 1)
		assert.False(t, isApproved)
	})

	t.Run("multiple_reviews_same_user", func(t *testing.T) {
		now := time.Now().Unix()

		reviews := []prp.Review{
			{ReviewState: ptrString("changes_requested"), SubmittedAt: ptrInt64(now), Reviewer: ptrString("user1")},
			{ReviewState: ptrString("approved"), SubmittedAt: ptrInt64(now + 1), Reviewer: ptrString("user1")},
		}

		isApproved, _ := isPrApproved(&reviews, 1)
		assert.True(t, isApproved)
	})
}

func TestIsPrReviewed(t *testing.T) {
	t.Run("min_reviews_required_zero", func(t *testing.T) {
		prDetails := &github.PullRequest{
			RequestedReviewers: []*github.User{},
		}
		isReviewed := isPrReviewed(0, prDetails)
		assert.True(t, isReviewed)
	})

	t.Run("no_requested_reviewers", func(t *testing.T) {
		prDetails := &github.PullRequest{
			RequestedReviewers: []*github.User{},
		}
		isReviewed := isPrReviewed(1, prDetails)
		assert.True(t, isReviewed)
	})

	t.Run("requested_reviewers_present", func(t *testing.T) {
		prDetails := &github.PullRequest{
			RequestedReviewers: []*github.User{
				{Login: github.String("user1")},
				{Login: github.String("user2")},
			},
		}
		isReviewed := isPrReviewed(1, prDetails)
		assert.False(t, isReviewed)
	})
}

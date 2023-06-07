package actor

import (
	"errors"
	"github.com/google/go-github/v52/github"
	"github.com/knadh/koanf/v2"
	prp "nudge/internal/database/pr"
	"nudge/internal/database/repository"
	provider "nudge/internal/provider/github"
	"sort"
)

type GithubUserName string

type ActorDetails struct {
	IsReviewer     bool
	GithubUserName GithubUserName
}

type ActorIdentifier interface {
	IdentifyActors(delayedPR prp.PRModel, repo repository.RepoModel, ko *koanf.Koanf) ([]ActorDetails, error)
}

type Actor struct{}

func (actor *Actor) IdentifyActors(delayedPR prp.PRModel, repo repository.RepoModel, ko *koanf.Koanf) ([]ActorDetails, error) {
	// Fetch the latest PR details from GitHub

	// Extract the reviewers in from the PR

	// If there are reviewers, then they directly become the blockers for the PR
	// Note: Once a requested reviewer submits a review, they are no longer considered a requested reviewer.
	// Reference: https://docs.github.com/en/rest/pulls/review-requests?apiVersion=2022-11-28#get-all-requested-reviewers-for-a-pull-request

	// If there are no reviewers, blocker could either be a reviewer or the author
	jwt, _ := provider.GenerateAppJWT(ko.String("app.private_key"), ko.String("github.app_id"))
	g := provider.Init(*jwt)
	iToken, appTokenErr := g.GetAppInstallationAccessToken(repo.InstallationId)
	if appTokenErr != nil {
		return nil, appTokenErr
	}

	g = provider.Init(*iToken.Token)
	prDetails, prErr := g.GetPrById(delayedPR.Number, repo.Owner, repo.Name)
	if prErr != nil {
		return nil, prErr
	}

	baseBranch := prDetails.Base.Ref
	protectionRules, protectionErr := g.GetBranchProtection(repo.Name, *baseBranch, repo.Owner)
	if protectionErr != nil {
		if !errors.Is(protectionErr, github.ErrBranchNotProtected) {
			return nil, protectionErr
		}
		// If the branch is not protected, move ahead
	}

	minReviewsRequired := 0
	if protectionRules != nil && protectionRules.RequiredPullRequestReviews != nil {
		minReviewsRequired = protectionRules.RequiredPullRequestReviews.RequiredApprovingReviewCount
	}

	prReviewed := isPrReviewed(minReviewsRequired, prDetails)
	if !prReviewed {
		actors := make([]ActorDetails, 0)
		for _, r := range prDetails.RequestedReviewers {
			login := GithubUserName(*r.Login)
			actors = append(actors, ActorDetails{
				IsReviewer:     true,
				GithubUserName: login,
			})
		}
		// Return the list of reviewers because of which the PR is blocked
		return actors, nil
	}

	prApproved, userReviewMap := isPrApproved(delayedPR.Reviews, minReviewsRequired)

	if prReviewed && prApproved {
		// return the author who now just needs to merge
		return []ActorDetails{{
			IsReviewer:     false,
			GithubUserName: GithubUserName(*prDetails.User.Login),
		}}, nil
	}

	pendingAuthorActItems := hasPendingActionItemsForAuthor(delayedPR.Reviews)
	if pendingAuthorActItems {
		// return author who might need to discuss with reviewer
		return []ActorDetails{{IsReviewer: false, GithubUserName: GithubUserName(*prDetails.User.Login)}}, nil
	} else {
		// return the reviewers
		actors := make([]ActorDetails, 0)
		for username, v := range userReviewMap {
			foundApproval := false
			for _, review := range v {
				if *review.ReviewState == "approved" {
					foundApproval = true
					break
				}
			}
			if !foundApproval {
				actors = append(actors, ActorDetails{
					IsReviewer:     true,
					GithubUserName: username,
				})
			}
		}

		if len(actors) == 0 {
			// If it was not able to identify any of the actor, default
			// it to the author
			// Note: This can happen when the minimum number of reviews required are more than 1
			// but there has not been any reviewer assigned
			actors = append(actors, ActorDetails{
				IsReviewer:     false,
				GithubUserName: GithubUserName(*prDetails.User.Login),
			})
		}
		return actors, nil
	}

}

// isPrReviewed Upon creating the pull request, authors typically add the reviewers
// that they would like to get a review from for the specific change.
// The reviewers are supposed to act on it and provide their comments.
// If the reviewers are not acting on the pull request after requesting a
// review, then the onus is going to be on the reviewers to act on the
// pull request and unblock it.
func isPrReviewed(minReviewsRequired int, prDetails *github.PullRequest) bool {
	reviewed := false
	if minReviewsRequired > 0 {
		if len(prDetails.RequestedReviewers) == 0 {
			// Since there are no pending reviewers on the PR
			// it has been reviewed
			reviewed = true
		}
	} else {
		if len(prDetails.RequestedReviewers) == 0 {
			// Since there are no minimum reviews required
			// and total reviewers are also zero PR state will be reviewed
			reviewed = true
		}
	}

	return reviewed
}

// isPrApproved A pull request is approved when the reviewers are satisfied with the changes
// and have no more comments or concerns about the change. The author can proceed to merge
// the change.
func isPrApproved(reviews *[]prp.Review, minReviewsRequired int) (bool, map[GithubUserName][]prp.Review) {
	approvals := make([]bool, 0)
	userReviewMap := make(map[GithubUserName][]prp.Review)

	if minReviewsRequired == 0 {
		// Since there are no minimum number of reviews required
		// the PR state will be approved
		approvals = append(approvals, true)
	} else if reviews == nil {
		// reviews are nil but min number of reviews required
		// are greater than 0
		approvals = append(approvals, false)
	} else if reviews != nil {
		if len(*reviews) < minReviewsRequired {
			// If the approvals received are less than
			// the minimum number of reviews required
			approvals = append(approvals, false)
		} else {
			for _, review := range *reviews {
				reviewer := GithubUserName(*review.Reviewer)
				_, exists := userReviewMap[reviewer]
				if exists {
					userReviewMap[reviewer] = append(userReviewMap[reviewer], review)
				} else {
					userReviewMap[reviewer] = []prp.Review{review}
				}
			}

			for _, v := range userReviewMap {
				sort.Sort(byReviewTime(v))
			}

			for _, v := range userReviewMap {
				for _, r := range v {
					if *r.ReviewState == "changes_requested" {
						approvals = append(approvals, false)
						break
					} else if *r.ReviewState == "approved" {
						approvals = append(approvals, true)
						break
					}
				}
			}
		}
	}

	approved := true
	for _, b := range approvals {
		if b == false {
			approved = false
			break
		}
	}
	return approved, userReviewMap
}

type byReviewTime []prp.Review

func (bt byReviewTime) Len() int {
	return len(bt)
}

func (bt byReviewTime) Swap(i, j int) {
	bt[i], bt[j] = bt[j], bt[i]
}

func (bt byReviewTime) Less(i, j int) bool {
	return *bt[i].SubmittedAt > *bt[j].SubmittedAt
}

// hasPendingActionItemsForAuthor The author has addressed the review comments, but the reviewer does
// not want to approve the changes, because they are not satisfied with the resolution
// provided by the author. These pull requests need further discussion
//
// Or the reviewer has left comments seeking some clarity or proposing recommendations.
// The author is responsible to address the review comments. Authors typically will
// have two choices: If they agree with the review comment, then they can resolve it,
// or if they disagree, then they can mark it as “won’t fix.” This condition is met if
// the author has review comments that need to be addressed..
func hasPendingActionItemsForAuthor(reviews *[]prp.Review) bool {
	pendingForAuthor := false

	if reviews != nil {
		for _, review := range *reviews {
			// Note: The review comments which are resolved have already been removed from
			// the review state (during the webhook event). This means that any other state
			// other than approved means that are some comments that need to be addressed.
			if *review.ReviewState != "approved" {
				pendingForAuthor = true
				break
			}
		}
	}

	return pendingForAuthor
}

package main

import (
	"github.com/google/go-github/v52/github"
	"github.com/labstack/echo/v4"
	"net/http"
	prp "nudge/internal/database/pr"
	"nudge/internal/database/repository"
	uc "nudge/internal/database/user"
	provider "nudge/internal/provider/github"
)

func handleWebhook(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)
	app.log.Println("Received webhook")

	payload, err := github.ValidatePayload(c.Request(), nil)
	if err != nil {
		return err
	}
	event, err := github.ParseWebHook(github.WebHookType(c.Request()), payload)
	if err != nil {
		return err
	}

	go func() {
		switch event := event.(type) {
		case *github.PullRequestEvent:
			handlePR(*event, app)
			break
		case *github.PullRequestReviewThreadEvent:
			updateWorkflow(*event, app)
			resolveReview(*event, app)
			break
		case *github.PullRequestReviewEvent:
			// https://docs.github.com/en/graphql/reference/enums#pullrequestreviewevent
			updateWorkflow(*event, app)
			addReview(*event, app)
			break
		case *github.PullRequestReviewCommentEvent:
			break
		case *github.InstallationEvent:
			uninstallApp(*event, app)
			break
		case *github.InstallationRepositoriesEvent:
			handleInstallRepositoryEvent(*event, app)
			break
		}
	}() // Process the webhook in a separate coroutine

	return c.JSON(http.StatusOK, okResp{"out"})
}

func handlePR(pr github.PullRequestEvent, app *App) {
	if pr.Action != nil {
		switch *pr.Action {
		case "opened":
			handleNewPRRequest(pr, app)
			break
		case "closed":
			handlePRCloseRequest(pr, app)
			updateWorkflow(pr, app)
			break
		case "reopened":
		case "ready_for_review":
			handlePRReopenRequest(pr, app)
			updateWorkflow(pr, app)
			break
		case "synchronize":
			updateWorkflow(pr, app)
			break
		case "review_requested":
			updateWorkflow(pr, app)
			updateReviewers(pr, app)
			break
		case "review_request_removed":
			updateReviewers(pr, app)
			break
		case "edited":
			break
		}
	}
}

func updateWorkflow(pr interface{}, app *App) {
	var (
		workflowLastActivity               int64
		workflowLastActionRecorded         string
		workflowLastActionCategoryRecorded string
		prId                               int64
	)

	switch pr.(type) {
	case github.PullRequestReviewThreadEvent:
		_pr := pr.(github.PullRequestReviewThreadEvent)
		workflowLastActivity = _pr.PullRequest.UpdatedAt.Unix()
		workflowLastActionRecorded = *_pr.Action
		workflowLastActionCategoryRecorded = prp.WorkflowActionTypeComment
		prId = *_pr.PullRequest.ID
		break
	case github.PullRequestEvent:
		_pr := pr.(github.PullRequestEvent)
		workflowLastActivity = _pr.PullRequest.UpdatedAt.Unix()
		workflowLastActionRecorded = *_pr.Action
		workflowLastActionCategoryRecorded = prp.WorkflowActionTypePull
		prId = *_pr.PullRequest.ID
		break
	case github.PullRequestReviewEvent:
		_pr := pr.(github.PullRequestReviewEvent)
		workflowLastActivity = _pr.PullRequest.UpdatedAt.Unix()
		workflowLastActionRecorded = *_pr.Action
		workflowLastActionCategoryRecorded = *_pr.Review.State
		prId = *_pr.PullRequest.ID
	}

	prModel := prp.Init(app.db)
	err := prModel.UpdateByPRId(prId, map[string]interface{}{
		"workflow_last_activity":                 workflowLastActivity,
		"last_workflow_action_recorded":          workflowLastActionRecorded,
		"last_workflow_action_category_recorded": workflowLastActionCategoryRecorded,
		//TODO: Better way to know the json name of the field in PRModel struct
	})

	if err != nil {
		app.log.Printf("Error while updating the workflow on PullRequestReviewThreadEvent %v", err)
	}

}

func handleNewPRRequest(pr github.PullRequestEvent, app *App) {
	prModel := prp.Init(app.db)
	model := prp.CreateDataModelForPR(*pr.PullRequest, *pr.Repo.ID)
	err := prModel.Create(model)
	if err != nil {
		app.log.Printf("Error while inserting a new PR record %v", err)
	}
}

func handlePRCloseRequest(pr github.PullRequestEvent, app *App) {
	prModel := prp.Init(app.db)
	err := prModel.UpdateByPRId(*pr.PullRequest.ID, map[string]interface{}{
		"status":        *pr.PullRequest.State,
		"pr_updated_at": pr.PullRequest.UpdatedAt.Unix(),
		//TODO: Better way to know the json name of the field in PRModel struct
	})
	if err != nil {
		app.log.Printf("Error while updating the PR status to closed %v", err)
	}
}

func handlePRReopenRequest(pr github.PullRequestEvent, app *App) {
	prModel := prp.Init(app.db)
	model := prp.CreateDataModelForPR(*pr.PullRequest, *pr.Repo.ID)
	err := prModel.Upsert(model)
	if err != nil {
		app.log.Printf("Error while carrying out the upsert operation for PR-Reopen event %v", err)
	}
}

func updateReviewers(pr github.PullRequestEvent, app *App) {
	if pr.RequestedReviewer != nil {
		prModel := prp.Init(app.db)
		reviewer := *pr.RequestedReviewer.Login
		removeReviewer := false
		if *pr.Action == "review_request_removed" {
			removeReviewer = true
		}
		err := prModel.UpdateReviewer(*pr.PullRequest.ID, reviewer, removeReviewer)
		if err != nil {
			lo.Printf("Failed to update reviewers for PR %d of repo %s - %v", *pr.Number, *pr.Repo.Name, err)
		}
	}
}

func addReview(pr github.PullRequestReviewEvent, app *App) {
	prModel := prp.Init(app.db)
	submittedAt := pr.Review.SubmittedAt.Unix()
	review := prp.Review{
		ReviewId:    *pr.Review.ID,
		ReviewState: pr.Review.State,
		Reviewer:    pr.Review.User.Login,
		SubmittedAt: &submittedAt,
	}
	err := prModel.UpdateReview(*pr.PullRequest.ID, review, false)
	if err != nil {
		lo.Printf("Failed to update review for PR %d of repo %s - %v", *pr.PullRequest.Number, *pr.Repo.Name, err)
	}
}

func resolveReview(pr github.PullRequestReviewThreadEvent, app *App) {
	if *pr.Action == "resolved" {
		prModel := prp.Init(app.db)
		if pr.Thread != nil {
			review := prp.Review{ReviewId: *pr.Thread.Comments[0].PullRequestReviewID}
			err := prModel.UpdateReview(*pr.PullRequest.ID, review, true)
			if err != nil {
				lo.Printf("Failed to remove review for PR %d of repo %s - %v", *pr.PullRequest.Number, *pr.Repo.Name, err)
			}
		}
	}
}

func uninstallApp(installation github.InstallationEvent, app *App) {
	if *installation.Action == "deleted" {
		uDelErr := uc.Init(app.db).Delete(*installation.Installation.ID)
		if uDelErr != nil {
			lo.Printf("Failed to delete user %v", uDelErr)
			return
		}
		repoDelErr := repository.Init(app.db).DeleteAll(*installation.Installation.ID)
		if repoDelErr != nil {
			lo.Printf("Failed to delete repository %v", repoDelErr)
			return
		}

		for _, repo := range installation.Repositories {
			prDelErr := prp.Init(app.db).DeleteAll(*repo.ID)
			if prDelErr != nil {
				lo.Printf("Failed to delete repository %s %v", *repo.Name, prDelErr)
			}
		}
	}
}

func handleInstallRepositoryEvent(installation github.InstallationRepositoriesEvent, app *App) {
	r := repository.Init(app.db)
	if *installation.Action == "added" {
		rModel := make([]repository.RepoModel, len(installation.RepositoriesAdded))
		for i, repo := range installation.RepositoriesAdded {
			rModel[i] = repository.RepoModel{
				InstallationId: *installation.Installation.ID,
				RepoId:         *repo.ID,
				Name:           *repo.Name,
				Owner:          *installation.Installation.Account.Login,
			}
		}
		err := r.Create(rModel)
		if err != nil {
			lo.Printf("Error while populating repos during the install repo event %v", err)
		}
		jwt, _ := provider.GenerateAppJWT(app.ko.String("app.private_key"), app.ko.String("github.app_id"))
		g := provider.Init(*jwt)
		iToken, appTokenErr := g.GetAppInstallationAccessToken(*installation.Installation.ID)
		if appTokenErr != nil {
			lo.Printf("Failed to fetch app access token while trying to add PRs %v", appTokenErr)
			return
		}

		// Populate all the PRs for the repositories added
		for _, inst := range installation.RepositoriesAdded {
			inst.Owner = installation.Installation.Account
			// The webhook does not send the owner information, which is required by
			// the populateActivePRs method
		}
		populateActivePRs(app, iToken.GetToken(), installation.RepositoriesAdded)
	} else if *installation.Action == "removed" {
		pr := prp.Init(app.db)
		for _, repo := range installation.RepositoriesRemoved {
			if repo != nil {
				// delete the repos
				rErr := r.DeleteOneById(*repo.ID)
				if rErr != nil {
					lo.Printf("Error removing the repository during the remove repository event %v", rErr)
				}
				// delete all the linked PRs
				err := pr.DeleteAll(*repo.ID)
				if err != nil {
					lo.Printf("Error removing PRs during the remove repository event %v", err)
				}
			}
		}
	}
}

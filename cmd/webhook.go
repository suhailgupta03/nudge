package main

import (
	"github.com/google/go-github/v51/github"
	"github.com/labstack/echo/v4"
	"net/http"
	prp "nudge/internal/database/pr"
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
	switch event := event.(type) {
	case *github.PullRequestEvent:
		handlePR(*event, app)
		break
	}
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
			break
		case "reopened":
			handlePRReopenRequest(pr, app)
			break
		case "edited":
			break
		}
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
		"status":        prp.PRStatusClosed,
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

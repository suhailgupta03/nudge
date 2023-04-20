package main

import (
	"fmt"
	"github.com/google/go-github/v51/github"
	"github.com/labstack/echo/v4"
	"net/http"
	prp "nudge/internal/database/pr"
	"nudge/prediction"
)

var (
	app *App
)

func handleWebhook(c echo.Context) error {
	app = c.Get("app").(*App)
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
		handlePR(*event)
	default:
		fmt.Println(event)
	}

	app.log.Println("Received webhook")
	return c.JSON(http.StatusOK, okResp{"out"})
}

func handlePR(pr github.PullRequestEvent) {
	if pr.Action != nil {
		switch *pr.Action {
		case "opened":
			handleNewPRRequest(pr)
			break
		case "closed":
			handlePRCloseRequest(pr)
			break
		case "reopened":
			handlePRReopenRequest(pr)
			break
		case "edited":
			break
		}
	}
}

func handleNewPRRequest(pr github.PullRequestEvent) {
	prModel := prp.Init(app.db)
	model := new(prp.PRModel)
	model.PRID = *pr.PullRequest.ID
	model.Number = *pr.Number
	model.RepoId = *pr.Repo.ID
	model.Status = prp.PRStatusOpen
	model.LifeTime = prediction.EstimateLifeTime()
	err := prModel.Create(model)
	if err != nil {
		app.log.Printf("Error while inserting a new PR record %v", err)
	}
}

func handlePRCloseRequest(pr github.PullRequestEvent) {
	prModel := prp.Init(app.db)
	err := prModel.UpdateByPRId(*pr.PullRequest.ID, map[string]string{
		"status": prp.PRStatusClosed,
		//TODO: Better way to know the json name of the field in PRModel struct
	})
	if err != nil {
		app.log.Printf("Error while updating the PR status to closed %v", err)
	}
}

func handlePRReopenRequest(pr github.PullRequestEvent) {
	prModel := prp.Init(app.db)
	model := new(prp.PRModel)
	model.PRID = *pr.PullRequest.ID
	model.Number = *pr.Number
	model.RepoId = *pr.Repo.ID
	model.Status = prp.PRStatusOpen
	model.LifeTime = prediction.EstimateLifeTime()
	err := prModel.Upsert(model)
	if err != nil {
		app.log.Printf("Error while carrying out the upsert operation for PRReopen event %v", err)
	}
}

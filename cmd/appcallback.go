package main

import (
	"github.com/google/go-github/v52/github"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/url"
	dbp "nudge/internal/database"
	prp "nudge/internal/database/pr"
	"nudge/internal/database/repository"
	uc "nudge/internal/database/user"
	"nudge/internal/provider/github"
	"strconv"
	"strings"
)

func handleGitHubAppCallback(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)
	code := c.QueryParam("code")
	installationId, _ := strconv.ParseInt(c.QueryParam("installation_id"), 10, 64)

	if len(code) == 0 {
		lo.Println("Got empty code in the callback. Something is not right.")

	} else {
		tokenDetails, err := provider.FetchGithubAppAccessToken(app.ko.String("github.client_id"), app.ko.String("github.client_secret"), code)
		if err != nil {
			app.log.Printf("Failed to fetch the acccess token %v\n", err)
			return err
		} else {
			/**
			Update the token details into the database
			*/
			g := provider.Init(tokenDetails.AccessToken)
			me, meErr := g.Me()
			if meErr != nil {
				lo.Fatalf("Failed to fetch user details from the oauth access token %v", meErr)
				return meErr
			}
			jwt, _ := provider.GenerateAppJWT(app.ko.String("app.private_key"), app.ko.String("github.app_id"))
			g = provider.Init(*jwt)
			iToken, appTokenErr := g.GetAppInstallationAccessToken(installationId)
			if appTokenErr != nil {

			}
			uCollection := uc.Init(app.db)
			uModel := new(uc.UserModel)
			uModel.GitHubUserId = *me.ID
			uModel.GitHubUserOauth = uc.GitHubOauthModel{
				GitHubAccessToken:  tokenDetails.AccessToken,
				GitHubRefreshToken: tokenDetails.RefreshToken,
			}
			uModel.GitHubApp = uc.GitHubAppModel{
				GitHubInstallationAccessToken: iToken.GetToken(),
				InstallationId:                installationId,
			}
			if me.Login != nil {
				uModel.GitHubUsername = *me.Login
			} else {
				// If we did not get the GitHub username
				// simply return
				return c.Redirect(http.StatusTemporaryRedirect, ko.String("server.ui")+"/?err=username_private")
			}

			if me.Email != nil {
				uModel.Email = *me.Email
			}

			uErr := uCollection.Create(uModel)
			if uErr != nil {
				writeException := uErr.(dbp.DatabaseException)
				if writeException.Code == 11000 {
					lo.Printf("User with email %s already exists with the system\n", uModel.Email)
				} else {
					lo.Printf("Failed to create the user %v", uErr)
					return uErr
				}
			}

			go populateReposToMonitor(app, uModel.GitHubApp.GitHubInstallationAccessToken, installationId)
		}
	}
	qp := url.Values{}
	qp.Set("installed", "true")
	return c.Redirect(http.StatusTemporaryRedirect, ko.String("server.ui")+"/?"+qp.Encode())
}

func populateReposToMonitor(app *App, appAccessToken string, installationId int64) {
	g := provider.Init(appAccessToken)
	repos, mErr := g.GetReposToMonitor()
	if mErr != nil {
		app.log.Printf("Failed to read repos to monitor %v", mErr)
	} else {
		r := repository.Init(app.db)
		rModel := make([]repository.RepoModel, len(repos))
		for i, item := range repos {
			rModel[i] = repository.RepoModel{
				InstallationId: installationId,
				RepoId:         *item.ID,
				Name:           *item.Name,
				Owner:          *item.Owner.Login,
			}
		}
		// Populate all repos linked to this app into the database
		err := r.Create(rModel)
		if err != nil {
			lo.Printf("Error while populating repos %v", err)
		}

		// Populate all open PRs into the database
		populateActivePRs(app, appAccessToken, repos)
	}
}

func populateActivePRs(app *App, appAccessToken string, repos []*github.Repository) {
	g := provider.Init(appAccessToken)
	prStateToFetch := "open"
	prModel := prp.Init(app.db)
	for _, repo := range repos {
		prs, prErr := g.GetPRs(*repo.Owner.Login, *repo.Name, &prStateToFetch)
		if prErr != nil {
			app.log.Printf("Failed to fetch PR details for repo %s %v", *repo.Name, prErr)
			continue
		}
		prModelList := make([]*prp.PRModel, 0)
		for _, pr := range prs {
			if app.ko.Bool("bot.ignore_bot_prs") {
				if pr.User != nil && pr.User.Type != nil && strings.ToLower(*pr.User.Type) == "bot" {
					// Ignore the PRs raised by bots!
					app.log.Printf("Ignoring the PR#%d for repo %s raised by bot", *pr.Number, *repo.Name)
					continue
				} else {
					// Since user is not defined (and its type is not known)
					// add to the PR list
					model := prp.CreateDataModelForPR(*pr, *repo.ID)
					prModelList = append(prModelList, model)
					if pr.User != nil && pr.User.Type != nil {
						app.log.Printf("User type detected as %s for PR#%d for repo %s", strings.ToLower(*pr.User.Type), *pr.Number, *repo.Name)
					}
				}
			} else {
				// Also include the PRs raised by the bots
				model := prp.CreateDataModelForPR(*pr, *repo.ID)
				prModelList = append(prModelList, model)
			}
		}
		bErr := prModel.BulkCreate(prModelList)
		if bErr != nil {
			app.log.Printf("Failed to insert open PR records for %s - %v", *repo.Name, bErr)
			continue
		}
	}
}

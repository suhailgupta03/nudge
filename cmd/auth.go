package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
	dbp "nudge/internal/database"
	uc "nudge/internal/database/user"
	provider "nudge/provider/github"
)

func handleGitHubOauth(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)
	code := c.QueryParam("code")
	if len(code) == 0 {
		lo.Fatalf("Did not find the code in query params during the oauth flow")
	} else {
		tokenDetails, err := provider.FetchOAuthAccessToken(app.ko.String("github.oauth_app_client_id"), app.ko.String("github.oauth_app_client_secret"), code)
		if err != nil {
			lo.Fatalf("Failed to fetch the token details %v", err)
			return err
		} else {
			// Use these token details to fetch the user information
			g := provider.Init(tokenDetails.AccessToken)
			me, meErr := g.Me()
			if meErr != nil {
				lo.Fatalf("Failed to fetch user details from the oauth access token %v", meErr)
				return meErr
			}
			uCollection := uc.Init(app.db)
			uModel := new(uc.UserModel)
			uModel.GitHubUserId = *me.ID
			uModel.GitHubUserOauth = uc.GitHubOauthModel{
				GitHubAccessToken: tokenDetails.AccessToken,
			}
			uModel.GitHubUsername = *me.Login
			uModel.Email = *me.Email

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
		}
	}
	return c.JSON(http.StatusOK, okResp{"out"})
}

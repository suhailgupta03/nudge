package main

import (
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/url"
	"nudge/internal/database/user"
	"strconv"
	"strings"
)

type slackAccessTokenResponse struct {
	Ok         bool   `json:"ok"`
	AppId      string `json:"app_id"`
	AuthedUser struct {
		Id string `json:"id"`
	} `json:"authed_user"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	BotUserId   string `json:"bot_user_id"`
	Team        struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	Enterprise          interface{} `json:"enterprise"`
	IsEnterpriseInstall bool        `json:"is_enterprise_install"`
}

type slackErrorResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

func handleSlackAuthRequest(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	code := c.QueryParam("code")
	if len(code) > 0 {
		form := url.Values{}
		form.Add("client_id", app.ko.String("slack.client_id"))
		form.Add("client_secret", app.ko.String("slack.client_secret"))
		form.Add("code", code)
		form.Add("grant_type", "authorization_code")

		req, _ := http.NewRequest("POST", "https://slack.com/api/oauth.v2.access", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			app.log.Printf("Failed to fetch the slack access token with status code %d ", resp.StatusCode)
			return errors.New("Failed with status code as " + strconv.Itoa(resp.StatusCode))
		}

		body, rErr := io.ReadAll(resp.Body)
		if rErr != nil {
			app.log.Printf("Failed to read the slack HTTP response body %v", rErr)
			return rErr
		}

		var jsonBody slackAccessTokenResponse
		json.Unmarshal(body, &jsonBody)
		if len(jsonBody.AccessToken) == 0 {
			var slackErrBody slackErrorResponse
			json.Unmarshal(body, &slackErrBody)
			return c.Redirect(http.StatusTemporaryRedirect, app.ko.String("server.ui")+"/slack-integration.html?ser="+slackErrBody.Error)
		} else {
			qp := url.Values{}
			qp.Set("sat", jsonBody.AccessToken)
			qp.Set("u", jsonBody.AuthedUser.Id)
			return c.Redirect(http.StatusTemporaryRedirect, app.ko.String("server.ui")+"/slack-integration.html?"+qp.Encode())
		}
	}

	return c.Redirect(http.StatusTemporaryRedirect, app.ko.String("server.ui")+"/slack-integration.html?ser=code_not_found")
}

type GitHubSlackMappingRequest struct {
	GitHubUserName   string `json:"git_hub_user_name"`
	SlackAccessToken string `json:"slack_access_token"`
	SlackUserId      string `json:"slack_user_id"`
}

type GitHubSlackMappingRequestAfterInstallation struct {
	GitHubUsername string `json:"git_hub_username"`
	SlackUserId    string `json:"slack_user_id"`
}

type CreateNewSlackUsers struct {
	InstallationId int64                                        `json:"installation_id"`
	Mapping        []GitHubSlackMappingRequestAfterInstallation `json:"mapping"`
}

func storeGitHubSlackMapping(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	var request GitHubSlackMappingRequest
	err := c.Bind(&request)
	if err != nil || len(request.GitHubUserName) == 0 || len(request.SlackAccessToken) == 0 {
		return c.String(http.StatusBadRequest, "bad request")
	}

	u := user.Init(app.db)
	err = u.UpdateSlackConfig(request.GitHubUserName, request.SlackAccessToken, request.SlackUserId)
	if err != nil {
		return c.JSON(http.StatusNotFound, "Please check if you have already installed the bot")
	} else {
		return c.JSON(http.StatusOK, "")
	}
}

// storeGitHubSlackMappingAfterInstallation use this method to populate new users
// for the slack workspace. This assumes that the GitHub and the slack installation is complete
// If the Slack installation is not complete when this method is called, the
// workflow will not start generating the Slack notifications
func storeGitHubSlackMappingAfterInstallation(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	var request CreateNewSlackUsers
	err := c.Bind(&request)
	if err != nil || len(request.Mapping) == 0 {
		return c.String(http.StatusBadRequest, "bad request")
	}

	u := user.Init(app.db)
	m := make([]user.GithubSlackMapping, 0)
	for _, rm := range request.Mapping {
		m = append(m, user.GithubSlackMapping{
			GitHubUsername: rm.GitHubUsername,
			SlackUserId:    rm.SlackUserId,
		})
	}

	err = u.CreateNewSlackUsers(request.InstallationId, m)
	if err != nil {
		return c.JSON(http.StatusNotFound, err.Error())
	} else {
		return c.JSON(http.StatusOK, "")
	}

}

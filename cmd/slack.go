package main

import (
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/url"
	"nudge/internal/database/user"
	"regexp"
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

type SlackGitHubMappingCommand struct {
	ApiAppId            string `form:"api_app_id"`
	ChannelId           string `form:"channel_id"`
	ChannelName         string `form:"channel_name"`
	Command             string `form:"command"`
	EnterpriseId        string `form:"enterprise_id,omitempty"`
	EnterpriseName      string `form:"enterprise_name,omitempty"`
	IsEnterpriseInstall bool   `form:"is_enterprise_install,omitempty"`
	ResponseUrl         string `form:"response_url"`
	TeamDomain          string `form:"team_domain,omitempty"`
	TeamId              string `form:"team_id"`
	Text                string `form:"text"`
	Token               string `form:"token"`
	TriggerId           string `form:"trigger_id"`
	UserId              string `form:"user_id"`
	UserName            string `form:"user_name"`
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

// / handleSlackMappingCommand maps the slack's user id with the github username
// example command /map-github installation-id github-username
func handleSlackMappingCommand(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
	)

	var request SlackGitHubMappingCommand
	err := c.Bind(&request)
	if err != nil || len(request.UserId) == 0 || len(request.Text) == 0 {
		return c.String(http.StatusBadRequest, "bad request")
	}

	re := regexp.MustCompile(`\s+`)
	commandSplit := re.Split(request.Text, -1)
	if len(commandSplit) != 2 {
		return c.String(http.StatusBadRequest, "Err..! Use command like this /map-github installation-id myGitHubUsername")
	}
	installationId, castErr := strconv.ParseInt(commandSplit[0], 10, 64)
	if castErr != nil {
		return c.String(http.StatusBadRequest, "Please check if the installation id is correct")
	}
	githubUserName := commandSplit[1]
	slackUserId := request.UserId

	u := user.Init(app.db)
	updateErr := u.CreateNewSlackUsers(installationId, []user.GithubSlackMapping{{GitHubUsername: githubUserName, SlackUserId: slackUserId}})
	if updateErr != nil {
		return c.JSON(http.StatusNotFound, err.Error())
	} else {
		return c.JSON(http.StatusOK, "Great! You will now start receiving the notifications")
	}
}

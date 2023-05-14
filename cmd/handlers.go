package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
)

type okResp struct {
	Data interface{} `json:"data"`
}

func handlePing(c echo.Context) error {
	return c.String(http.StatusOK, "pong!")
}

func initHTTPHandlers(e *echo.Echo, app *App) {
	var g *echo.Group
	g = e.Group("")

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:  "static",
		Index: "index.html",
	}))

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		// Generic, non-echo error. Log it.
		if _, ok := err.(*echo.HTTPError); !ok {
			app.log.Println(err.Error())
		}
		e.DefaultHTTPErrorHandler(err, c)
	}

	e.GET("/ping", handlePing)

	// Public Endpoints For GitHub Callbacks
	g.GET("/github/app/callback", handleGitHubAppCallback)
	g.POST("/github/app/webhook", handleWebhook)
	g.GET("/github/oauth/callback", handleGitHubOauth)

	// Public Endpoints for Slack Callbacks
	g.GET("/slack/auth", handleSlackAuthRequest)
	g.POST("/slack/github", storeGitHubSlackMapping)
	g.POST("/slack/command/map-github", handleSlackMappingCommand)
	// the following endpoint is internal [does not use auth as of today]
	g.POST("/slack/users", storeGitHubSlackMappingAfterInstallation)
}

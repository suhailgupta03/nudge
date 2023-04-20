package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type okResp struct {
	Data interface{} `json:"data"`
}

func initHTTPHandlers(e *echo.Echo, app *App) {
	var g *echo.Group
	g = e.Group("")

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		// Generic, non-echo error. Log it.
		if _, ok := err.(*echo.HTTPError); !ok {
			app.log.Println(err.Error())
		}
		e.DefaultHTTPErrorHandler(err, c)
	}

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Up and running")
	})

	// Public Endpoints For GitHub Callbacks
	g.GET("/github/app/callback", handleGitHubAppCallback)
	g.POST("/github/app/webhook", handleWebhook)
	g.GET("/github/oauth/callback", handleGitHubOauth)
}

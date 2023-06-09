package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"strconv"
	"time"
)

type GithubAppTokenDetails struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type OAuthAppTokenDetails struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type GitHub struct {
	ctx    context.Context
	client *github.Client
}

func GenerateAppJWT(key string, appId string) (*string, error) {
	// https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/about-authentication-with-a-github-app#authenticating-as-a-github-app
	//https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation
	// https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-json-web-token-jwt-for-a-github-app
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": appId,
		"exp": time.Now().Add(time.Minute * 10).Unix(),
		"alg": "RS256",
		"iat": time.Now().Add(-time.Second).Unix(),
	})

	privateKey, pkErr := jwt.ParseRSAPrivateKeyFromPEM([]byte(key))
	if pkErr != nil {
		return nil, pkErr
	}
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(privateKey)
	return &tokenString, err
}

func Init(token string) *GitHub {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &GitHub{
		client: client,
		ctx:    ctx,
	}
}

// Me https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28#get-the-authenticated-user
func (g *GitHub) Me() (*github.User, error) {
	me, _, err := g.client.Users.Get(g.ctx, "")
	return me, err
}

func (g *GitHub) GetAppInstallationAccessToken(installationId int64) (*github.InstallationToken, error) {
	token, _, err := g.client.Apps.CreateInstallationToken(g.ctx, installationId, nil)
	return token, err
}

// GetReposToMonitor https://docs.github.com/en/rest/apps/installations?apiVersion=2022-11-28#list-repositories-accessible-to-the-app-installation
func (g *GitHub) GetReposToMonitor() ([]*github.Repository, error) {

	repos, repoResponse, err := g.client.Apps.ListRepos(g.ctx, nil)

	reposList := make([]*github.Repository, 0)
	if err != nil {
		return nil, err
	}

	reposList = repos.Repositories
	if repoResponse.NextPage > 0 {
		next := repoResponse.NextPage
		for {
			r, rr, rerr := g.client.Apps.ListRepos(g.ctx, &github.ListOptions{
				Page: next,
			})

			for _, repo := range r.Repositories {
				reposList = append(reposList, repo)
			}

			if rerr != nil || rr.NextPage == 0 || len(r.Repositories) == 0 {
				if rerr != nil {
					err = rerr
				}
				break
			}

			next = rr.NextPage
		}
	}

	return reposList, err
}

func (g *GitHub) GetPrById(prNumber int, owner, repo string) (*github.PullRequest, error) {
	pr, _, err := g.client.PullRequests.Get(g.ctx, owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (g *GitHub) GetPRs(owner, repoName string, state *string) ([]*github.PullRequest, error) {
	prList, prr, err := g.client.PullRequests.List(g.ctx, owner, repoName, nil)
	if err != nil {
		return nil, err
	}

	for i := prr.NextPage; i > 0; {
		prl, r, perr := g.client.PullRequests.List(g.ctx, owner, repoName, &github.PullRequestListOptions{
			ListOptions: github.ListOptions{
				Page: i,
			},
		})
		if perr != nil || len(prl) == 0 {
			if perr != nil {
				return nil, perr
			}
			break
		}
		for _, _r := range prl {
			prList = append(prList, _r)
		}
		i = r.NextPage
	}

	if state == nil {
		return prList, nil
	}

	prs := make([]*github.PullRequest, 0)

	for _, pr := range prList {
		if *pr.State == *state {
			prs = append(prs, pr)
		}
	}

	return prs, err
}

func (g *GitHub) GetBranchProtection(repo, branch, owner string) (*github.Protection, error) {
	protection, _, err := g.client.Repositories.GetBranchProtection(g.ctx, owner, repo, branch)
	return protection, err
}

func (g *GitHub) PostComment(repo, owner string, prNumber int, body string) error {
	_, _, err := g.client.Issues.CreateComment(g.ctx, owner, repo, prNumber, &github.IssueComment{
		Body: &body,
	})
	return err
}

func fetchAccessToken(clientId, clientSecret, code string) ([]byte, error) {
	postBody, _ := json.Marshal(map[string]string{
		"client_id":     clientId,
		"client_secret": clientSecret,
		"code":          code,
	})
	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(postBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("Failed with status code as " + strconv.Itoa(resp.StatusCode))
	}

	body, _ := io.ReadAll(resp.Body)
	return body, nil
}

func FetchGithubAppAccessToken(clientId, clientSecret, code string) (*GithubAppTokenDetails, error) {
	t, err := fetchAccessToken(clientId, clientSecret, code)
	if err != nil {
		return nil, err
	}
	var jsonBody GithubAppTokenDetails
	json.Unmarshal(t, &jsonBody)

	return &jsonBody, nil
}

func FetchOAuthAccessToken(clientId, clientSecret, code string) (*OAuthAppTokenDetails, error) {
	t, err := fetchAccessToken(clientId, clientSecret, code)
	if err != nil {
		return nil, err
	}
	var jsonBody OAuthAppTokenDetails
	json.Unmarshal(t, &jsonBody)

	return &jsonBody, nil
}

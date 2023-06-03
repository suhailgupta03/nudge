package user

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"nudge/internal/database"
	"time"
)

type GitHubAppModel struct {
	GitHubInstallationAccessToken string `bson:"git_hub_installation_access_token" json:"git_hub_installation_access_token"`
	InstallationId                int64  `bson:"installation_id" json:"installation_id"`
	UpdatedAt                     int64  `bson:"updated_at" json:"updated_at"`
}

type GitHubOauthModel struct {
	GitHubAccessToken  string `bson:"git_hub_access_token" json:"git_hub_access_token"`
	GitHubRefreshToken string `bson:"git_hub_refresh_token,omitempty" json:"git_hub_refresh_token,omitempty"`
	UpdatedAt          int64  `bson:"updated_at" json:"updated_at"`
}

type NotificationBusinessHours struct {
	StartHours int `json:"start_time" bson:"start_time"`
	EndHours   int `json:"end_time" bson:"end_time"`
}

type TimeZone string

type UserModel struct {
	GitHubUsername   string           `bson:"git_hub_username" json:"git_hub_username"`
	GitHubUserId     int64            `bson:"git_hub_user_id" json:"git_hub_user_id"`
	Email            string           `bson:"email" json:"email"`
	GitHubUserOauth  GitHubOauthModel `bson:"git_hub_user_oauth" json:"git_hub_user_oauth"`
	GitHubApp        GitHubAppModel   `bson:"git_hub_app" json:"git_hub_app"`
	SlackAccessToken *string          `json:"slack_access_token,omitempty" bson:"slack_access_token,omitempty"`
	// SlackUserId Can be one of public / private channel id or user id, depending on the use-case
	SlackUserId *string `json:"slack_user_id,omitempty" bson:"slack_user_id,omitempty"`
	// GithubSlackMapping can be kept nil if the messages must always be sent to a channel
	GithubSlackMapping *[]GithubSlackMapping      `json:"github_slack_mapping,omitempty" bson:"github_slack_mapping,omitempty"`
	TimeZone           *TimeZone                  `json:"time_zone,omitempty" bson:"time_zone,omitempty"`
	BusinessHours      *NotificationBusinessHours `json:"business_hours,omitempty" bson:"business_hours,omitempty"`
	CreatedAt          int64                      `bson:"created_at" json:"created_at"`
	UpdatedAt          int64                      `bson:"updated_at" json:"updated_at"`
}

type GithubSlackMapping struct {
	GitHubUsername string `bson:"git_hub_username" json:"git_hub_username"`
	SlackUserId    string `bson:"slack_user_id" json:"slack_user_id"`
}
type User struct {
	Collection *mongo.Collection
}

type UserTimezoneService interface {
	FindUserTimezoneByInstallationId(installationId int64) (*TimeZone, *NotificationBusinessHours, error)
}

func Init(db *mongo.Database) *User {
	return &User{
		Collection: db.Collection(database.UserCollection),
	}
}

func (u *User) Create(user *UserModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ts := time.Now().Unix()
	user.CreatedAt = ts
	user.UpdatedAt = ts
	user.GitHubApp.UpdatedAt = ts
	user.GitHubUserOauth.UpdatedAt = ts

	_, err := u.Collection.InsertOne(ctx, user)
	if err != nil {
		return database.ParseDatabaseError(err)
	}

	return nil
}

func (u *User) Delete(installationId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := map[string]int64{
		"git_hub_app.installation_id": installationId,
	}
	_, err := u.Collection.DeleteOne(ctx, where)
	return err
}

func (u *User) UpdateSlackConfig(githubUserName, token, slackUserId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := map[string]string{
		"git_hub_username": githubUserName,
	}

	toUpdate := map[string]interface{}{
		"$set": map[string]interface{}{
			"slack_access_token": token,
			"slack_user_id":      slackUserId,
			"updated_at":         time.Now().Unix(),
		},
	}
	r := u.Collection.FindOneAndUpdate(ctx, where, toUpdate, nil)
	return r.Err()
}

func (u *User) CreateNewSlackUsers(installationId int64, mapping []GithubSlackMapping) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := map[string]interface{}{
		"git_hub_app.installation_id": installationId,
	}

	toUpdate := map[string]interface{}{
		"$addToSet": map[string]interface{}{
			"github_slack_mapping": map[string][]GithubSlackMapping{
				"$each": mapping,
			},
		},
		"$set": map[string]interface{}{
			"updated_at": time.Now().Unix(),
		},
	}

	r := u.Collection.FindOneAndUpdate(ctx, where, toUpdate, nil)
	return r.Err()
}
func (u *User) FindUserByGitHubUsername(githubUserName string, installationId int64) (*UserModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := map[string]interface{}{
		"$or": []map[string]interface{}{
			{"git_hub_username": githubUserName},
			{"github_slack_mapping": map[string]interface{}{
				"$elemMatch": map[string]string{
					"git_hub_username": githubUserName,
				},
			}},
		},
		"git_hub_app.installation_id": installationId,
	}

	r := u.Collection.FindOne(ctx, where)
	if r.Err() != nil {
		return nil, r.Err()
	}
	var user UserModel
	r.Decode(&user)

	return &user, nil
}

func (u *User) FindSlackUserIdFromInstallationId(installationId int64) (*UserModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := map[string]interface{}{
		"git_hub_app.installation_id": installationId,
		"slack_user_id": map[string]interface{}{
			"$exists": true,
		},
	}

	r := u.Collection.FindOne(ctx, where)
	if r.Err() != nil {
		return nil, r.Err()
	}
	var user UserModel
	r.Decode(&user)

	return &user, nil
}

func (u *User) FindUserTimezoneByInstallationId(installationId int64) (*TimeZone, *NotificationBusinessHours, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	where := map[string]int64{
		"git_hub_app.installation_id": installationId,
	}
	result := u.Collection.FindOne(ctx, where, nil)
	if result.Err() != nil {
		return nil, nil, result.Err()
	}
	var U UserModel
	result.Decode(&U)

	return U.TimeZone, U.BusinessHours, nil
}

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

type UserModel struct {
	GitHubUsername  string           `bson:"git_hub_username" json:"git_hub_username"`
	GitHubUserId    int64            `bson:"git_hub_user_id" json:"git_hub_user_id"`
	Email           string           `bson:"email" json:"email"`
	GitHubUserOauth GitHubOauthModel `bson:"git_hub_user_oauth" json:"git_hub_user_oauth"`
	GitHubApp       GitHubAppModel   `bson:"git_hub_app" json:"git_hub_app"`
	CreatedAt       int64            `bson:"created_at" json:"created_at"`
	UpdatedAt       int64            `bson:"updated_at" json:"updated_at"`
}

type User struct {
	Collection *mongo.Collection
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

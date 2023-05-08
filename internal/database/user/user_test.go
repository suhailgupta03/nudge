package user

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"os"
	"testing"
	"time"
)

var dbTest *mongo.Database

// setUp is called to initialize the test database.
func setUp() {
	mongodbURI := os.Getenv("MONGODB_URI_TEST")
	testDb := os.Getenv("TEST_DB_NAME")
	if mongodbURI == "" {
		mongodbURI = "mongodb://localhost:27017/test_nudge" // Replace with your MongoDB test instance URI
	}

	if testDb == "" {
		testDb = "test_nudge"
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(mongodbURI))
	if err != nil {
		fmt.Println("Cannot create Mongo client", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		fmt.Println("Cannot connect to Mongo", err)
		os.Exit(1)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println("Cannot ping Mongo", err)
		os.Exit(1)
	}

	dbTest = client.Database(testDb)
}

// tearDown is called to clean up the test database.
func tearDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = dbTest.Drop(ctx)
}

func TestUser_Create_Delete(t *testing.T) {
	setUp()
	defer tearDown()

	u := Init(dbTest)
	userModel := &UserModel{
		GitHubUsername: "testuser",
		GitHubUserId:   12345,
		Email:          "testuser@example.com",
	}

	err := u.Create(userModel)
	if err != nil {
		t.Errorf("Cannot create user: %v", err)
	}

	err = u.Delete(userModel.GitHubUserId)
	if err != nil {
		t.Errorf("Cannot delete user: %v", err)
	}
}

func TestUser_FindUserByGitHubUsername(t *testing.T) {
	setUp()
	defer tearDown()

	u := Init(dbTest)
	userModel := &UserModel{
		GitHubUsername: "testuser",
		GitHubUserId:   12345,
		Email:          "testuser@example.com",
	}

	err := u.Create(userModel)
	if err != nil {
		t.Errorf("Cannot create user: %v", err)
	}

	foundUser, err := u.FindUserByGitHubUsername(userModel.GitHubUsername)
	if err != nil {
		t.Errorf("Cannot find user by GitHub username: %v", err)
	}

	if foundUser.GitHubUsername != userModel.GitHubUsername {
		t.Errorf("Found user GitHubUsername does not match the created user")
	}
}

func TestUser_UpdateSlackConfig(t *testing.T) {
	setUp()
	defer tearDown()

	u := Init(dbTest)
	userModel := &UserModel{
		GitHubUsername:   "testuser",
		GitHubUserId:     12345,
		Email:            "testuser@example.com",
		SlackAccessToken: nil,
		SlackUserId:      nil,
	}

	err := u.Create(userModel)
	if err != nil {
		t.Errorf("Cannot create user: %v", err)
	}

	err = u.UpdateSlackConfig(userModel.GitHubUsername, "test_token", "U123456")
	if err != nil {
		t.Errorf("Cannot update Slack config: %v", err)
	}

	updatedUser, err := u.FindUserByGitHubUsername(userModel.GitHubUsername)
	if err != nil {
		t.Errorf("Cannot find user by GitHub username: %v", err)
	}

	if *updatedUser.SlackAccessToken != "test_token" || *updatedUser.SlackUserId != "U123456" {
		t.Errorf("Slack config not updated correctly")
	}
}

package user

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
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

type findUserByGitHubUsernameTestCase struct {
	Name           string
	GitHubUsername string
	InstallationId int64
	SetupUser      *UserModel
	ExpectError    bool
}

func TestUser_FindUserByGitHubUsername(t *testing.T) {
	setUp()
	defer tearDown()

	u := Init(dbTest)
	testCases := []findUserByGitHubUsernameTestCase{
		{
			Name:           "Case 1: Find user with valid Git Hub username and InstallationId",
			GitHubUsername: "user1",
			InstallationId: 12345,
			SetupUser: &UserModel{
				GitHubUsername: "user1",
				Email:          "user1@example.com",
				GitHubApp:      GitHubAppModel{InstallationId: 12345},
			},
			ExpectError: false,
		},
		{
			Name:           "Case 2: No user found with provided GitHub username and InstallationId",
			GitHubUsername: "nonexistent_user",
			InstallationId: 12345,
			SetupUser:      nil,
			ExpectError:    true,
		},
		{
			Name:           "Case 3: Find user by matching GitHub username in GitHubSlackMapping",
			GitHubUsername: "user2",
			InstallationId: 12345,
			SetupUser: &UserModel{
				GitHubUsername: "other_user",
				Email:          "other@example.com",
				GitHubApp:      GitHubAppModel{InstallationId: 12345},
				GithubSlackMapping: &[]GithubSlackMapping{
					{GitHubUsername: "user2", SlackUserId: "U123"},
				},
			},
			ExpectError: false,
		},
		{
			Name:           "Case 4: User not found due to GitHub username mismatch",
			GitHubUsername: "nonexistent_user",
			InstallationId: 12345,
			SetupUser: &UserModel{
				GitHubUsername: "user3",
				Email:          "user3@example.com",
				GitHubApp:      GitHubAppModel{InstallationId: 12345},
			},
			ExpectError: true,
		},
		{
			Name:           "Case 5: User not found due to InstallationId mismatch",
			GitHubUsername: "user4",
			InstallationId: 67890,
			SetupUser: &UserModel{
				GitHubUsername: "user4",
				Email:          "user4@example.com",
				GitHubApp:      GitHubAppModel{InstallationId: 11111},
			},
			ExpectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {

			if testCase.SetupUser != nil {
				err := u.Create(testCase.SetupUser)
				require.NoErrorf(t, err, "Failed to set up initial data for: %s", testCase.Name)
			}

			user, err := u.FindUserByGitHubUsername(testCase.GitHubUsername, testCase.InstallationId)

			if testCase.ExpectError {
				require.Errorf(t, err, "Expected an error in %s", testCase.Name)
			} else {
				require.NoErrorf(t, err, "Unexpected error in %s", testCase.Name)
				require.NotNilf(t, user, "Expected a user to be found in %s", testCase.Name)
			}
		})
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
		GitHubApp: GitHubAppModel{
			InstallationId: 123,
		},
	}

	err := u.Create(userModel)
	if err != nil {
		t.Errorf("Cannot create user: %v", err)
	}

	err = u.UpdateSlackConfig(userModel.GitHubUsername, "test_token", "U123456")
	if err != nil {
		t.Errorf("Cannot update Slack config: %v", err)
	}

	updatedUser, err := u.FindUserByGitHubUsername(userModel.GitHubUsername, userModel.GitHubApp.InstallationId)
	if err != nil {
		t.Errorf("Cannot find user by GitHub username: %v", err)
	}

	if *updatedUser.SlackAccessToken != "test_token" || *updatedUser.SlackUserId != "U123456" {
		t.Errorf("Slack config not updated correctly")
	}
}

type testCreateNewSlackUsersTestCase struct {
	Name                     string
	GitHubUsername           string
	InstallationId           int64
	SetupUser                *UserModel
	Mapping                  []GithubSlackMapping
	ExpectedAfterMappingUser *UserModel
	ExpectError              bool
}

func TestCreateNewSlackUsers(t *testing.T) {
	setUp()
	defer tearDown()

	testCases := []testCreateNewSlackUsersTestCase{
		{
			Name:                     "Case 1: Add new GitHubSlackMapping",
			InstallationId:           12345,
			SetupUser:                &UserModel{GitHubUsername: "user1", Email: "user1@example.com", GitHubApp: GitHubAppModel{InstallationId: 12345}},
			Mapping:                  []GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}},
			ExpectedAfterMappingUser: &UserModel{GitHubUsername: "user1", Email: "user1@example.com", GitHubApp: GitHubAppModel{InstallationId: 12345}, GithubSlackMapping: &[]GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}}},
			ExpectError:              false,
		},
		{
			Name:           "Case 2: InstallationId mismatch",
			InstallationId: 67890,
			SetupUser:      &UserModel{GitHubUsername: "user3", Email: "user3@example.com", GitHubApp: GitHubAppModel{InstallationId: 12345}},
			Mapping:        []GithubSlackMapping{{GitHubUsername: "user4", SlackUserId: "U222"}},
			ExpectError:    true,
		},
		{
			Name:                     "Case 3: Add multiple GitHubSlackMappings",
			InstallationId:           12345,
			SetupUser:                &UserModel{GitHubUsername: "user1", Email: "user1@example.com", GitHubApp: GitHubAppModel{InstallationId: 12345}},
			Mapping:                  []GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}, {GitHubUsername: "user5", SlackUserId: "U555"}},
			ExpectedAfterMappingUser: &UserModel{GitHubUsername: "user1", Email: "user1@example.com", GitHubApp: GitHubAppModel{InstallationId: 12345}, GithubSlackMapping: &[]GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}, {GitHubUsername: "user5", SlackUserId: "U555"}}},
			ExpectError:              false,
		},
		{
			Name:                     "Case 4: Add existing mapping",
			InstallationId:           1234567,
			SetupUser:                &UserModel{GitHubUsername: "user11", Email: "user11@example.com", GitHubApp: GitHubAppModel{InstallationId: 1234567}, GithubSlackMapping: &[]GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}}},
			Mapping:                  []GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}},
			ExpectedAfterMappingUser: &UserModel{GitHubUsername: "user11", Email: "user11@example.com", GitHubApp: GitHubAppModel{InstallationId: 1234567}, GithubSlackMapping: &[]GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}}},
			ExpectError:              false,
		},
		{
			Name:                     "Case 5: Add a new mapping to an existing user with multiple mappings",
			InstallationId:           1234599,
			SetupUser:                &UserModel{GitHubUsername: "user101", Email: "user1@example.com", GitHubApp: GitHubAppModel{InstallationId: 1234599}, GithubSlackMapping: &[]GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}, {GitHubUsername: "user5", SlackUserId: "U555"}}},
			Mapping:                  []GithubSlackMapping{{GitHubUsername: "user6", SlackUserId: "U666"}},
			ExpectedAfterMappingUser: &UserModel{GitHubUsername: "user101", Email: "user1@example.com", GitHubApp: GitHubAppModel{InstallationId: 1234599}, GithubSlackMapping: &[]GithubSlackMapping{{GitHubUsername: "user2", SlackUserId: "U111"}, {GitHubUsername: "user5", SlackUserId: "U555"}, {GitHubUsername: "user6", SlackUserId: "U666"}}},
			ExpectError:              false,
		},
	}

	userRepo := Init(dbTest)

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {

			if testCase.SetupUser != nil {
				err := userRepo.Create(testCase.SetupUser)
				require.NoErrorf(t, err, "Failed to set up initial data for: %s", testCase.Name)
			}

			err := userRepo.CreateNewSlackUsers(testCase.InstallationId, testCase.Mapping)

			if testCase.ExpectError {
				require.Errorf(t, err, "Expected an error in %s", testCase.Name)
			} else {
				require.NoErrorf(t, err, "Unexpected error in %s", testCase.Name)

				foundUser, _ := userRepo.FindUserByGitHubUsername(testCase.SetupUser.GitHubUsername, testCase.InstallationId)
				require.NotNil(t, foundUser)
				require.Equal(t, testCase.ExpectedAfterMappingUser.GitHubUsername, foundUser.GitHubUsername)
				require.Equal(t, testCase.ExpectedAfterMappingUser.Email, foundUser.Email)
				require.Equal(t, testCase.ExpectedAfterMappingUser.GitHubApp.InstallationId, foundUser.GitHubApp.InstallationId)

				if testCase.ExpectedAfterMappingUser.GithubSlackMapping != nil && foundUser.GithubSlackMapping != nil {
					require.ElementsMatch(t, *testCase.ExpectedAfterMappingUser.GithubSlackMapping, *foundUser.GithubSlackMapping)
				}
			}
		})
	}
}

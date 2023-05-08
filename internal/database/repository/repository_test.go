package repository

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
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
	if mongodbURI == "" {
		mongodbURI = "mongodb://localhost:27017/test_repository" // Replace with your MongoDB test instance URI
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

	dbTest = client.Database("test_repository") // Replace 'test_repository' with your test database name
}

// tearDown is called to clean up the test database.
func tearDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = dbTest.Drop(ctx)
}

func TestRepository_Create(t *testing.T) {
	setUp()
	defer tearDown()

	repo := Init(dbTest)
	repoModel := []RepoModel{
		{
			InstallationId: 1,
			RepoId:         1,
			Name:           "test-repo-1",
			Owner:          "test-owner",
		},
		{
			InstallationId: 2,
			RepoId:         2,
			Name:           "test-repo-2",
			Owner:          "test-owner",
		},
	}

	err := repo.Create(repoModel)
	if err != nil {
		t.Errorf("Cannot create repositories: %v", err)
	}
}

func TestRepository_GetAll(t *testing.T) {
	setUp()
	defer tearDown()

	repo := Init(dbTest)
	repoModel := []RepoModel{
		{
			InstallationId: 1,
			RepoId:         1,
			Name:           "test-repo-1",
			Owner:          "test-owner",
		},
		{
			InstallationId: 2,
			RepoId:         2,
			Name:           "test-repo-2",
			Owner:          "test-owner",
		},
	}

	err := repo.Create(repoModel)
	if err != nil {
		t.Errorf("Cannot create repositories: %v", err)
	}

	results, err := repo.GetAll()
	if err != nil {
		t.Errorf("Cannot get all repositories: %v", err)
	}

	if len(*results) != 2 {
		t.Errorf("GetAll should return an array of length 2")
	}
}

func TestRepository_DeleteAll(t *testing.T) {
	setUp()
	defer tearDown()

	repo := Init(dbTest)
	repoModel := []RepoModel{
		{
			InstallationId: 1,
			RepoId:         1,
			Name:           "test-repo-1",
			Owner:          "test-owner",
		},
		{
			InstallationId: 1,
			RepoId:         2,
			Name:           "test-repo-2",
			Owner:          "test-owner",
		},
	}

	err := repo.Create(repoModel)
	if err != nil {
		t.Errorf("Cannot create repositories: %v", err)
	}

	err = repo.DeleteAll(1)
	if err != nil {
		t.Errorf("Cannot delete all repositories: %v", err)
	}

	results, err := repo.GetAll()
	if err != nil {
		t.Errorf("Cannot get all repositories: %v", err)
	}

	if len(*results) != 0 {
		t.Errorf("DeleteAll should remove all repositories with InstallationId 1")
	}
}

func TestRepository_DeleteOne(t *testing.T) {
	setUp()
	defer tearDown()

	repo := Init(dbTest)
	repoModel := []RepoModel{
		{
			InstallationId: 1,
			RepoId:         1,
			Name:           "test-repo-1",
			Owner:          "test-owner-1",
		},
		{
			InstallationId: 2,
			RepoId:         2,
			Name:           "test-repo-2",
			Owner:          "test-owner-2",
		},
	}

	err := repo.Create(repoModel)
	if err != nil {
		t.Errorf("Cannot create repositories: %v", err)
	}

	err = repo.DeleteOne(1)
	if err != nil {
		t.Errorf("Cannot delete one repository: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := bson.M{
		"installation_id": int64(1),
	}
	var foundRepo RepoModel
	err = repo.Collection.FindOne(ctx, where).Decode(&foundRepo)
	if err == nil {
		t.Errorf("DeleteOne should have removed the repository with InstallationId 1")
	}
}

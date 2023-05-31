package pr

import (
	"context"
	"fmt"
	"github.com/google/go-github/v52/github"
	"github.com/stretchr/testify/assert"
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
		mongodbURI = "mongodb://localhost:27017/test_pr" // Replace with your MongoDB test instance URI
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

	dbTest = client.Database("test_pr") // Replace 'test_pr' with your test database name
}

// tearDown is called to clean up the test database.
func tearDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = dbTest.Drop(ctx)
}

func TestPR_Create(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)
	prModel := &PRModel{
		Number: 1,
		PRID:   1,
		RepoId: 1,
		Status: "open",
	}

	err := prRepo.Create(prModel)
	if err != nil {
		t.Errorf("Cannot create PR: %v", err)
	}
}

func TestPR_BulkCreate(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)

	prModels := []*PRModel{
		{
			Number: 1,
			PRID:   1,
			RepoId: 1,
			Status: "open",
		},
		{
			Number: 2,
			PRID:   2,
			RepoId: 1,
			Status: "open",
		},
	}

	err := prRepo.BulkCreate(prModels)
	if err != nil {
		t.Errorf("Cannot bulk create PRs: %v", err)
	}
}

func TestPR_UpdateByPRId(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)
	prModel := &PRModel{
		Number: 1,
		PRID:   1,
		RepoId: 1,
		Status: "open",
	}

	err := prRepo.Create(prModel)
	if err != nil {
		t.Errorf("Cannot create PR: %v", err)
	}

	toUpdate := map[string]interface{}{
		"status": "closed",
	}

	err = prRepo.UpdateByPRId(1, toUpdate)
	if err != nil {
		t.Errorf("Cannot update PR by PRID: %v", err)
	}
}

func TestPR_UpdateReviewer(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)
	prModel := &PRModel{
		Number: 1,
		PRID:   1,
		RepoId: 1,
		Status: "open",
	}

	err := prRepo.Create(prModel)
	if err != nil {
		t.Errorf("Cannot create PR: %v", err)
	}

	err = prRepo.UpdateReviewer(1, "test_reviewer", false)
	if err != nil {
		t.Errorf("Cannot update reviewer: %v", err)
	}
}

func TestPR_UpdateReview(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)
	prModel := &PRModel{
		Number: 1,
		PRID:   1,
		RepoId: 1,
		Status: "open",
	}

	err := prRepo.Create(prModel)
	if err != nil {
		t.Errorf("Cannot create PR: %v", err)
	}

	review := Review{
		ReviewId:    1,
		ReviewState: nil,
		Reviewer:    nil,
		SubmittedAt: nil,
	}

	err = prRepo.UpdateReview(1, review, false)
	if err != nil {
		t.Errorf("Cannot update review: %v", err)
	}
}

func TestPR_Upsert(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)
	prModel := &PRModel{
		Number: 1,
		PRID:   1,
		RepoId: 1,
		Status: "open",
	}

	// Upsert should create a new PR because it doesn't exist
	err := prRepo.Upsert(prModel)
	if err != nil {
		t.Errorf("Cannot upsert PR (create): %v", err)
	}

	// Update the PR model
	prModel.Status = "closed"

	// Upsert should update the existing PR
	err = prRepo.Upsert(prModel)
	if err != nil {
		t.Errorf("Cannot upsert PR (update): %v", err)
	}
}

func TestPR_DeleteAll(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)
	prModel := &PRModel{
		Number: 1,
		PRID:   1,
		RepoId: 1,
		Status: "open",
	}

	err := prRepo.Create(prModel)
	if err != nil {
		t.Errorf("Cannot create PR: %v", err)
	}

	err = prRepo.DeleteAll(1)
	if err != nil {
		t.Errorf("Cannot delete all PRs: %v", err)
	}
}

func TestCreateDataModelForPR(t *testing.T) {
	setUp()
	defer tearDown()

	exampleCreatedAt := time.Now()
	exampleUpdatedAt := exampleCreatedAt.Add(2 * time.Hour)

	ghPr := &github.PullRequest{
		ID:        github.Int64(1),
		Number:    github.Int(1),
		State:     github.String("open"),
		CreatedAt: &github.Timestamp{Time: exampleCreatedAt},
		UpdatedAt: &github.Timestamp{Time: exampleUpdatedAt},
	}

	prModel := CreateDataModelForPR(*ghPr, 1)

	if prModel.PRID != *ghPr.ID || prModel.Number != *ghPr.Number ||
		prModel.RepoId != 1 || prModel.Status != *ghPr.State ||
		prModel.PRCreatedAt != ghPr.CreatedAt.Unix() || prModel.PRUpdatedAt != ghPr.UpdatedAt.Unix() {
		t.Errorf("CreateDataModelForPR did not create the correct PRModel")
	}
}

func TestIncrementTotalCommentsMade(t *testing.T) {
	setUp()
	defer tearDown()

	prRepo := Init(dbTest)

	testCases := []struct {
		description      string
		inputPRID        int64
		createPR         bool
		initialComments  int
		expectedComments int
		expectedError    error
	}{
		{
			description:      "Increment total bot comments when a PRModel exists",
			inputPRID:        123456,
			createPR:         true,
			initialComments:  0,
			expectedComments: 1,
			expectedError:    nil,
		},
		{
			description:      "Increment total bot comments when a PRModel exists",
			inputPRID:        123457,
			createPR:         true,
			initialComments:  5,
			expectedComments: 6,
			expectedError:    nil,
		},
		{
			description:   "Increment total bot comments when a PRModel does not exist",
			inputPRID:     999999,
			createPR:      false,
			expectedError: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			if testCase.createPR {
				pr := &PRModel{
					PRID:             testCase.inputPRID,
					RepoId:           654321,
					Status:           "open",
					TotalBotComments: new(int),
					PRCreatedAt:      time.Now().Unix(),
					PRUpdatedAt:      time.Now().Unix(),
					CreatedAt:        time.Now().Unix(),
					UpdatedAt:        time.Now().Unix(),
				}
				if testCase.initialComments > 0 {
					*pr.TotalBotComments = testCase.initialComments
				}
				err := prRepo.Create(pr)
				if err != nil {
					t.Fatal("Failed to create test PR:", err)
				}
				defer prRepo.Collection.DeleteOne(context.Background(), map[string]interface{}{"prid": testCase.inputPRID})
				testCase.inputPRID = pr.PRID
			}

			err := prRepo.IncrementTotalCommentsMade(testCase.inputPRID)

			assert.Equal(t, testCase.expectedError, err)

			if err == nil && testCase.createPR {
				var updatedPR PRModel
				err = prRepo.Collection.FindOne(context.Background(), map[string]interface{}{"prid": testCase.inputPRID}).Decode(&updatedPR)
				if err != nil {
					t.Fatal("Failed to retrieve updated PRModel:", err)
				}
				assert.Equal(t, testCase.expectedComments, *updatedPR.TotalBotComments)
			}
		})
	}
}

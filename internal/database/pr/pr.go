package pr

import (
	"context"
	"errors"
	"github.com/google/go-github/v52/github"
	"go.mongodb.org/mongo-driver/mongo"
	"nudge/internal/database"
	"nudge/prediction"
	"time"
)

const (
	WorkFlowStateActive = iota
	WorkFlowStateTerminated
)

const (
	WorkflowActionTypeComment = "comment"
	WorkflowActionTypePull    = "pull"
)

type PRModel struct {
	Number                             int       `json:"number" bson:"number"`
	PRID                               int64     `json:"prid" bson:"prid"`
	RepoId                             int64     `json:"repo_id" bson:"repo_id"`
	Status                             string    `json:"status" bson:"status"`
	LifeTime                           int       `json:"life_time" bson:"life_time"`
	WorkflowState                      int       `json:"workflow_state" bson:"workflow_state"`
	WorkflowLastActivity               *int64    `json:"workflow_last_activity,omitempty" bson:"workflow_last_activity,omitempty"`
	LastWorkflowActionRecorded         *string   `json:"last_workflow_action_recorded,omitempty" bson:"last_workflow_action_recorded,omitempty"`
	LastWorkflowActionCategoryRecorded *string   `json:"last_workflow_action_category_recorded,omitempty" bson:"last_workflow_action_category_recorded,omitempty"`
	RequestedReviewers                 *[]string `json:"requested_reviewers,omitempty" bson:"requested_reviewers,omitempty"`
	Reviews                            *[]Review `json:"reviews,omitempty" bson:"reviews,omitempty"`
	PRCreatedAt                        int64     `json:"pr_created_at" bson:"pr_created_at"`
	PRUpdatedAt                        int64     `json:"pr_updated_at" bson:"pr_updated_at"`
	CreatedAt                          int64     `bson:"created_at" json:"created_at" bson:"created_at"`
	UpdatedAt                          int64     `bson:"updated_at" json:"updated_at" bson:"updated_at"`
}

type Review struct {
	ReviewId    int64   `json:"review_id" bson:"review_id"`
	ReviewState *string `json:"review_state,omitempty" bson:"review_state,omitempty"`
	Reviewer    *string `json:"reviewer,omitempty" bson:"reviewer,omitempty"`
	SubmittedAt *int64  `json:"submitted_at,omitempty" bson:"submitted_at,omitempty"`
}

type PR struct {
	Collection *mongo.Collection
}

func Init(db *mongo.Database) *PR {
	return &PR{
		Collection: db.Collection(database.PRCollection),
	}
}

func (pr *PR) GetOpenPRs(repoId int64) (*[]PRModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	where := make(map[string]interface{})
	where["status"] = "open"
	where["repo_id"] = repoId
	cursor, err := pr.Collection.Find(ctx, where, nil)
	if err != nil {
		return nil, err
	}
	var results []PRModel
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	return &results, nil
}

func (pr *PR) Create(prm *PRModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ts := time.Now().Unix()
	prm.CreatedAt = ts
	prm.UpdatedAt = ts

	_, err := pr.Collection.InsertOne(ctx, prm)
	if err != nil {
		return database.ParseDatabaseError(err)
	}

	return nil
}

func (pr *PR) BulkCreate(prms []*PRModel) error {
	if len(prms) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	prmsToCreate := make([]interface{}, len(prms))
	for i, prm := range prms {
		ts := time.Now().Unix()
		prm.CreatedAt = ts
		prm.UpdatedAt = ts
		prmsToCreate[i] = prm
	}
	_, err := pr.Collection.InsertMany(ctx, prmsToCreate, nil)
	return err
}

func (pr *PR) UpdateByPRId(prId int64, toUpdate interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := make(map[string]int64)
	where["prid"] = prId
	ts := time.Now().Unix()
	var toUpdateWithOperator map[string]interface{}

	switch toUpdate.(type) {
	case *PRModel:
		updateModel := toUpdate.(*PRModel)
		updateModel.UpdatedAt = ts
		toUpdateWithOperator = map[string]interface{}{
			"$set": updateModel,
		}
		break
	case map[string]interface{}:
		updateModel := toUpdate.(map[string]interface{})
		updateModel["updated_at"] = ts
		toUpdateWithOperator = map[string]interface{}{
			"$set": updateModel,
		}
		break
	}

	if toUpdateWithOperator != nil {
		_, err := pr.Collection.UpdateOne(ctx, where, toUpdateWithOperator, nil)
		return err
	} else {
		return errors.New("could not handle the type while updating by PR ID")
	}
}

func (pr *PR) UpdateReviewer(prId int64, reviewer string, remove bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := make(map[string]int64)
	where["prid"] = prId
	ts := time.Now().Unix()

	toUpdate := make(map[string]interface{})

	if remove {
		toPull := make(map[string]string)
		toPull["requested_reviewers"] = reviewer
		toUpdate["$pull"] = toPull
	} else {
		toPush := make(map[string]string)
		toPush["requested_reviewers"] = reviewer
		toUpdate["$addToSet"] = toPush
	}
	toUpdate["$set"] = map[string]interface{}{
		"updated_at": ts,
	}
	_, err := pr.Collection.UpdateOne(ctx, where, toUpdate)
	return err
}

func (pr *PR) UpdateReview(prId int64, review Review, remove bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := make(map[string]int64)
	where["prid"] = prId
	ts := time.Now().Unix()

	toUpdate := make(map[string]interface{})

	if remove {
		toPull := make(map[string]Review)
		toPull["reviews"] = Review{
			ReviewId: review.ReviewId,
		}
		toUpdate["$pull"] = toPull
	} else {
		toPush := make(map[string]Review)
		toPush["reviews"] = review
		toUpdate["$addToSet"] = toPush
	}
	toUpdate["$set"] = map[string]interface{}{
		"updated_at": ts,
	}
	_, err := pr.Collection.UpdateOne(ctx, where, toUpdate)
	return err
}

func (pr *PR) Upsert(prm *PRModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var sResult PRModel

	err := pr.Collection.FindOne(ctx, map[string]interface{}{
		"prid": prm.PRID,
	}, nil).Decode(&sResult)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Insert a new record
			return pr.Create(prm)
		}
		return err
	}

	// If the document exists, update the document
	return pr.UpdateByPRId(prm.PRID, prm)

}

func (pr *PR) DeleteAll(repoId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	where := map[string]int64{
		"repo_id": repoId,
	}
	_, err := pr.Collection.DeleteMany(ctx, where)
	return err
}

func CreateDataModelForPR(pr github.PullRequest, repoId int64) *PRModel {
	model := new(PRModel)
	model.PRID = *pr.ID
	model.Number = *pr.Number
	model.RepoId = repoId
	model.Status = *pr.State
	model.PRCreatedAt = pr.CreatedAt.Unix()
	model.PRUpdatedAt = pr.UpdatedAt.Unix()
	model.LifeTime = prediction.EstimateLifeTime()
	model.WorkflowState = WorkFlowStateActive
	if len(pr.RequestedReviewers) > 0 {
		reviewers := make([]string, 0)
		for _, r := range pr.RequestedReviewers {
			if r.Login != nil {
				reviewers = append(reviewers, *r.Login)
			}
		}
		model.RequestedReviewers = &reviewers
	}
	// Update with the reviewers if there is any
	return model
}

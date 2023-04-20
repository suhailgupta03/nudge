package pr

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"nudge/internal/database"
	"reflect"
	"time"
)

const (
	PRStatusOpen   = "open"
	PRStatusClosed = "closed"
)

type PRModel struct {
	Number    int    `json:"number"`
	PRID      int64  `json:"prid"`
	RepoId    int64  `json:"repo_id"`
	Status    string `json:"status"`
	LifeTime  int    `json:"life_time"`
	CreatedAt int64  `bson:"created_at" json:"created_at"`
	UpdatedAt int64  `bson:"updated_at" json:"updated_at"`
}

type PR struct {
	Collection *mongo.Collection
}

func Init(db *mongo.Database) *PR {
	return &PR{
		Collection: db.Collection(database.PRCollection),
	}
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

	fmt.Println(reflect.TypeOf(toUpdate))
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

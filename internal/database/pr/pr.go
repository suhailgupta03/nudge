package pr

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"nudge/internal/database"
	"time"
)

const (
	PRStatusOpen   = "open"
	PRStatusClosed = "closed"
)

type PRModel struct {
	Number   int    `json:"number"`
	PRID     int64  `json:"prid"`
	RepoId   int64  `json:"repo_id"`
	Status   string `json:"status"`
	LifeTime int    `json:"life_time"`
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
		prmsToCreate[i] = prm
	}
	_, err := pr.Collection.InsertMany(ctx, prmsToCreate, nil)
	if err != nil {
		return database.ParseDatabaseError(err)
	}

	return nil
}

func (pr *PR) UpdateByPRId(prId int64, toUpdate interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	where := make(map[string]int64)
	where["prid"] = prId
	toUpdateWithOperator := map[string]interface{}{
		"$set": toUpdate,
	}
	_, err := pr.Collection.UpdateOne(ctx, where, toUpdateWithOperator, nil)
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

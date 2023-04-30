package repository

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"nudge/internal/database"
	"time"
)

type RepoModel struct {
	InstallationId int64  `bson:"installation_id" json:"installation_id"`
	RepoId         int64  `bson:"repo_id" json:"repo_id"`
	Name           string `bson:"name" json:"name"`
	Owner          string `bson:"owner" json:"owner"`
	CreatedAt      int64  `bson:"created_at" json:"created_at"`
	UpdatedAt      int64  `bson:"updated_at" json:"updated_at"`
}

type Repository struct {
	Collection *mongo.Collection
}

func Init(db *mongo.Database) *Repository {
	return &Repository{
		Collection: db.Collection(database.RepositoryCollection),
	}
}

func (repo *Repository) Create(r []RepoModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	records := make([]interface{}, len(r))
	for i, item := range r {
		ts := time.Now().Unix()
		item.CreatedAt = ts
		item.UpdatedAt = ts
		records[i] = item
	}

	_, err := repo.Collection.InsertMany(ctx, records)
	return err
}

func (repo *Repository) GetAll() (*[]RepoModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cursor, err := repo.Collection.Find(ctx, bson.D{}, nil)
	if err != nil {
		return nil, err
	}
	var results []RepoModel
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	return &results, nil
}

func (repo *Repository) DeleteAll(installationId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	where := map[string]int64{
		"installation_id": installationId,
	}
	_, err := repo.Collection.DeleteMany(ctx, where)
	return err
}

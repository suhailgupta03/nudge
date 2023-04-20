package repository

import (
	"context"
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

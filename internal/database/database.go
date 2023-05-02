package database

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type DatabaseException struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (de DatabaseException) Error() string {
	if de.Name != "" {
		return fmt.Sprintf("(%v) %v", de.Name, de.Message)
	}
	return de.Message
}

const (
	UserCollection       = "user"
	RepositoryCollection = "repositories"
	PRCollection         = "pr"
)

var availableCollections = []string{
	UserCollection,
	RepositoryCollection,
	PRCollection,
}

var indexDetails = map[string][]mongo.IndexModel{
	UserCollection: {
		{Keys: bson.D{{"git_hub_username", 1}}},
		{Keys: bson.D{{"git_hub_user_id", 1}}},
		{Keys: bson.D{{"email", 1}}},
		{Keys: bson.D{{"git_hub_app.installation_id", 1}}, Options: options.Index().SetUnique(true)},
	},
	RepositoryCollection: {
		{Keys: bson.D{{"repo_id", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"installation_id", 1}}},
	},
	PRCollection: {
		{Keys: bson.D{{"repo_id", 1}}},
		{Keys: bson.D{{"number", 1}}},
		{Keys: bson.D{{"prid", 1}}, Options: options.Index().SetUnique(true)},
	},
}

func SyncIndexes(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, collection := range availableCollections {
		_, has := indexDetails[collection]
		if has {
			db.Collection(collection).Indexes().CreateMany(ctx, indexDetails[collection])
		}
	}
}

func ParseDatabaseError(err error) error {
	mdException := err.(mongo.WriteException)
	dException := new(DatabaseException)
	if len(mdException.WriteErrors) > 0 {
		wErr := mdException.WriteErrors[0]
		dException.Message = wErr.Message
		dException.Name = wErr.Details.String()
		dException.Code = wErr.Code
	}
	return *dException
}

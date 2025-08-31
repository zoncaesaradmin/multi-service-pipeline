package configstore

import (
	"context"
)

// ConfigStore defines an interface for a key-value config/document store, similar to MongoDB
// Methods are modeled after common MongoDB operations

type ConfigStore interface {
	InsertOne(ctx context.Context, collection string, document interface{}) (insertedID interface{}, err error)
	FindOne(ctx context.Context, collection string, filter interface{}, result interface{}) error
	FindMany(ctx context.Context, collection string, filter interface{}, results interface{}) error
	UpdateOne(ctx context.Context, collection string, filter interface{}, update interface{}) error
	DeleteOne(ctx context.Context, collection string, filter interface{}) error
	CountDocuments(ctx context.Context, collection string, filter interface{}) (int64, error)
	Close() error
}

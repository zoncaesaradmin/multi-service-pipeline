//go:build !local
// +build !local

package configstore

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfigStore struct {
	client *mongo.Client
	dbName string
}

func NewMongoConfigStore(uri, dbName string) (ConfigStore, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &MongoConfigStore{client: client, dbName: dbName}, nil
}

func (m *MongoConfigStore) InsertOne(ctx context.Context, collection string, document interface{}) (interface{}, error) {
	coll := m.client.Database(m.dbName).Collection(collection)
	res, err := coll.InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}
	return res.InsertedID, nil
}

func (m *MongoConfigStore) FindOne(ctx context.Context, collection string, filter interface{}, result interface{}) error {
	coll := m.client.Database(m.dbName).Collection(collection)
	return coll.FindOne(ctx, filter).Decode(result)
}

func (m *MongoConfigStore) FindMany(ctx context.Context, collection string, filter interface{}, results interface{}) error {
	coll := m.client.Database(m.dbName).Collection(collection)
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	return cursor.All(ctx, results)
}

func (m *MongoConfigStore) UpdateOne(ctx context.Context, collection string, filter interface{}, update interface{}) error {
	coll := m.client.Database(m.dbName).Collection(collection)
	_, err := coll.UpdateOne(ctx, filter, update)
	return err
}

func (m *MongoConfigStore) DeleteOne(ctx context.Context, collection string, filter interface{}) error {
	coll := m.client.Database(m.dbName).Collection(collection)
	_, err := coll.DeleteOne(ctx, filter)
	return err
}

func (m *MongoConfigStore) CountDocuments(ctx context.Context, collection string, filter interface{}) (int64, error) {
	coll := m.client.Database(m.dbName).Collection(collection)
	return coll.CountDocuments(ctx, filter)
}

func (m *MongoConfigStore) Close() error {
	return m.client.Disconnect(context.Background())
}

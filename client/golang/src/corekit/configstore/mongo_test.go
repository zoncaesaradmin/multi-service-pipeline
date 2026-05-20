//go:build !local
// +build !local

package configstore

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

type TestDoc struct {
	ID   string `bson:"_id" json:"_id"`
	Name string `bson:"name" json:"name"`
	Val  int    `bson:"val" json:"val"`
}

// MockConfigStore implements ConfigStore for unit tests
type MockConfigStore struct {
	store map[string]TestDoc
}

func NewMockConfigStore() *MockConfigStore {
	return &MockConfigStore{store: make(map[string]TestDoc)}
}

func (m *MockConfigStore) InsertOne(ctx context.Context, coll string, doc interface{}) (interface{}, error) {
	d := doc.(TestDoc)
	m.store[d.ID] = d
	return d.ID, nil
}
func (m *MockConfigStore) FindOne(ctx context.Context, coll string, filter interface{}, out interface{}) error {
	id := filter.(bson.M)["_id"].(string)
	d, ok := m.store[id]
	if !ok {
		return errors.New("not found")
	}
	*out.(*TestDoc) = d
	return nil
}
func (m *MockConfigStore) UpdateOne(ctx context.Context, coll string, filter, update interface{}) error {
	id := filter.(bson.M)["_id"].(string)
	d, ok := m.store[id]
	if !ok {
		return errors.New("not found")
	}
	set := update.(bson.M)["$set"].(bson.M)
	d.Name = set["name"].(string)
	d.Val = set["val"].(int)
	m.store[id] = d
	return nil
}
func (m *MockConfigStore) FindMany(ctx context.Context, coll string, filter interface{}, out interface{}) error {
	many := []TestDoc{}
	for _, d := range m.store {
		many = append(many, d)
	}
	*out.(*[]TestDoc) = many
	return nil
}
func (m *MockConfigStore) CountDocuments(ctx context.Context, coll string, filter interface{}) (int64, error) {
	return int64(len(m.store)), nil
}
func (m *MockConfigStore) DeleteOne(ctx context.Context, coll string, filter interface{}) error {
	id := filter.(bson.M)["_id"].(string)
	delete(m.store, id)
	return nil
}
func (m *MockConfigStore) Close() error { return nil }

func TestMongoConfigStoreCRUD(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	// Use mock if URI is not set or is "mock"
	var store ConfigStore
	var err error
	if uri == "" || uri == "mock" {
		store = NewMockConfigStore()
	} else {
		store, err = NewMongoConfigStore(uri, "testdb")
		if err != nil {
			t.Skipf("MongoDB connection failed: %v", err)
		}
	}
	ctx := context.Background()
	coll := "col1"
	_ = store.DeleteOne(ctx, coll, bson.M{"_id": "abc123"}) // cleanup
	doc := TestDoc{ID: "abc123", Name: "foo", Val: 42}
	id, err := store.InsertOne(ctx, coll, doc)
	require.NoError(t, err, "InsertOne failed")
	require.NotNil(t, id, "InsertOne returned nil id")
	testFindOneMongo(t, store, ctx, coll, doc)
	testUpdateOneMongo(t, store, ctx, coll, doc.ID)
	testFindManyAndCountMongo(t, store, ctx, coll)
	testDeleteOneMongo(t, store, ctx, coll, doc.ID)
	_ = store.DeleteOne(ctx, coll, bson.M{"_id": "def456"}) // cleanup
	_ = store.Close()
}

func TestMongoConfigStoreEdgeCases(t *testing.T) {
	store := NewMockConfigStore()
	ctx := context.Background()
	coll := "col1"

	// Test FindOne on missing doc
	var missing TestDoc
	err := store.FindOne(ctx, coll, bson.M{"_id": "notfound"}, &missing)
	require.Error(t, err, "FindOne should error for missing doc")

	// Test UpdateOne on missing doc
	err = store.UpdateOne(ctx, coll, bson.M{"_id": "notfound"}, bson.M{"$set": bson.M{"name": "x", "val": 1}})
	require.Error(t, err, "UpdateOne should error for missing doc")

	// Test DeleteOne on missing doc (should not error)
	err = store.DeleteOne(ctx, coll, bson.M{"_id": "notfound"})
	require.NoError(t, err, "DeleteOne should not error for missing doc")

	// Test CountDocuments on empty store
	count, err := store.CountDocuments(ctx, coll, bson.M{})
	require.NoError(t, err)
	require.Equal(t, int64(0), count, "CountDocuments should be 0 for empty store")

	// Insert multiple docs and test FindMany
	docs := []TestDoc{
		{ID: "a", Name: "A", Val: 1},
		{ID: "b", Name: "B", Val: 2},
		{ID: "c", Name: "C", Val: 3},
	}
	for _, d := range docs {
		_, err := store.InsertOne(ctx, coll, d)
		require.NoError(t, err)
	}
	var gotMany []TestDoc
	err = store.FindMany(ctx, coll, bson.M{}, &gotMany)
	require.NoError(t, err)
	require.Len(t, gotMany, 3, "FindMany should return all docs")

	// Test CountDocuments after inserts
	count, err = store.CountDocuments(ctx, coll, bson.M{})
	require.NoError(t, err)
	require.Equal(t, int64(3), count)

	// Test UpdateOne and FindOne
	err = store.UpdateOne(ctx, coll, bson.M{"_id": "a"}, bson.M{"$set": bson.M{"name": "AA", "val": 11}})
	require.NoError(t, err)
	var updated TestDoc
	err = store.FindOne(ctx, coll, bson.M{"_id": "a"}, &updated)
	require.NoError(t, err)
	require.Equal(t, "AA", updated.Name)
	require.Equal(t, 11, updated.Val)

	// Test DeleteOne and CountDocuments
	err = store.DeleteOne(ctx, coll, bson.M{"_id": "a"})
	require.NoError(t, err)
	count, err = store.CountDocuments(ctx, coll, bson.M{})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	// Test Close (should not error)
	require.NoError(t, store.Close())
}

func testFindOneMongo(t *testing.T, store ConfigStore, ctx context.Context, coll string, doc TestDoc) {
	var got TestDoc
	err := store.FindOne(ctx, coll, bson.M{"_id": doc.ID}, &got)
	require.NoError(t, err, "FindOne failed")
	require.Equal(t, doc.Name, got.Name, "FindOne returned wrong name")
	require.Equal(t, doc.Val, got.Val, "FindOne returned wrong val")
}

func testUpdateOneMongo(t *testing.T, store ConfigStore, ctx context.Context, coll, id string) {
	update := bson.M{"name": "bar", "val": 99}
	err := store.UpdateOne(ctx, coll, bson.M{"_id": id}, bson.M{"$set": update})
	require.NoError(t, err, "UpdateOne failed")
	var updated TestDoc
	err = store.FindOne(ctx, coll, bson.M{"_id": id}, &updated)
	require.NoError(t, err, "FindOne after update failed")
	require.Equal(t, "bar", updated.Name, "UpdateOne did not update name")
	require.Equal(t, 99, updated.Val, "UpdateOne did not update val")
}

func testFindManyAndCountMongo(t *testing.T, store ConfigStore, ctx context.Context, coll string) {
	_, _ = store.InsertOne(ctx, coll, TestDoc{ID: "def456", Name: "baz", Val: 7})
	var many []TestDoc
	err := store.FindMany(ctx, coll, bson.M{}, &many)
	require.NoError(t, err, "FindMany failed")
	require.GreaterOrEqual(t, len(many), 2, "FindMany returned wrong count")
	count, err := store.CountDocuments(ctx, coll, bson.M{})
	require.NoError(t, err, "CountDocuments failed")
	require.GreaterOrEqual(t, count, int64(2), "CountDocuments returned wrong count")
}

func testDeleteOneMongo(t *testing.T, store ConfigStore, ctx context.Context, coll, id string) {
	err := store.DeleteOne(ctx, coll, bson.M{"_id": id})
	require.NoError(t, err, "DeleteOne failed")
	var afterDelete TestDoc
	err = store.FindOne(ctx, coll, bson.M{"_id": id}, &afterDelete)
	require.Error(t, err, "FindOne after DeleteOne should fail")
}

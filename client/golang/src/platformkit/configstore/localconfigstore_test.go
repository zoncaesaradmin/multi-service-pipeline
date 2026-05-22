//go:build local
// +build local

package configstore

import (
	"context"
	"os"
	"reflect"
	"testing"
)

type TestDoc struct {
	ID   string `bson:"_id" json:"_id"`
	Name string `bson:"name" json:"name"`
	Val  int    `bson:"val" json:"val"`
}

func TestLocalFileConfigStoreCRUD(t *testing.T) {
	filePath := "test_configstore.json"
	_ = os.Remove(filePath)
	store, err := NewLocalFileConfigStore(filePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	ctx := context.Background()
	doc := TestDoc{ID: "abc123", Name: "foo", Val: 42}
	id, err := store.InsertOne(ctx, "col1", doc)
	if err != nil {
		t.Fatalf("InsertOne failed: %v", err)
	}
	if id != doc.ID {
		t.Errorf("InsertOne returned wrong id: got %v, want %v", id, doc.ID)
	}
	var got TestDoc
	if err := store.FindOne(ctx, "col1", map[string]interface{}{"_id": doc.ID}, &got); err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}
	if !reflect.DeepEqual(got, doc) {
		t.Errorf("FindOne returned wrong doc: got %+v, want %+v", got, doc)
	}
	// Update
	update := map[string]interface{}{"name": "bar", "val": 99}
	if err := store.UpdateOne(ctx, "col1", map[string]interface{}{"_id": doc.ID}, update); err != nil {
		t.Fatalf("UpdateOne failed: %v", err)
	}
	var updated TestDoc
	if err := store.FindOne(ctx, "col1", map[string]interface{}{"_id": doc.ID}, &updated); err != nil {
		t.Fatalf("FindOne after update failed: %v", err)
	}
	if updated.Name != "bar" || updated.Val != 99 {
		t.Errorf("UpdateOne did not update fields: got %+v", updated)
	}
	// FindMany
	_, _ = store.InsertOne(ctx, "col1", TestDoc{ID: "def456", Name: "baz", Val: 7})
	var many []TestDoc
	if err := store.FindMany(ctx, "col1", map[string]interface{}{}, &many); err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}
	if len(many) != 2 {
		t.Errorf("FindMany returned wrong count: got %d, want 2", len(many))
	}
	// CountDocuments
	count, err := store.CountDocuments(ctx, "col1", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CountDocuments failed: %v", err)
	}
	if count != 2 {
		t.Errorf("CountDocuments returned wrong count: got %d, want 2", count)
	}
	// DeleteOne
	if err := store.DeleteOne(ctx, "col1", map[string]interface{}{"_id": doc.ID}); err != nil {
		t.Fatalf("DeleteOne failed: %v", err)
	}
	var afterDelete TestDoc
	if err := store.FindOne(ctx, "col1", map[string]interface{}{"_id": doc.ID}, &afterDelete); err == nil {
		t.Errorf("FindOne after DeleteOne should fail, got doc: %+v", afterDelete)
	}
	_ = os.Remove(filePath)
}

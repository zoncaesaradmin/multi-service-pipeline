//go:build local
// +build local

package datastore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

const testFilePath = "test_localdatastore.json"

type testDoc struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
	Val  int    `json:"val"`
}

func TestLocalIndexAndGetFileBased(t *testing.T) {
	_ = os.Remove(testFilePath)
	client := NewLocalDatastoreClient(testFilePath)
	ctx := context.Background()
	index := "testindex"
	id := "1"
	doc := testDoc{ID: id, Name: "testname", Val: 42}
	if err := client.Index(ctx, index, id, doc); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	var got testDoc
	if err := client.Get(ctx, index, id, &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != doc.Name || got.Val != doc.Val {
		t.Errorf("Get returned wrong doc: %+v", got)
	}
	_ = client.Delete(ctx, index, id)
	_ = os.Remove(testFilePath)
}

func TestLocalBulkIndexAndSearchFileBased(t *testing.T) {
	_ = os.Remove(testFilePath)
	client := NewLocalDatastoreClient(testFilePath)
	ctx := context.Background()
	index := "testbulk"
	docs := []BulkDoc{
		{ID: "a", Body: testDoc{ID: "a", Name: "A", Val: 1}, Action: "index"},
		{ID: "b", Body: testDoc{ID: "b", Name: "B", Val: 2}, Action: "index"},
		{ID: "c", Body: testDoc{ID: "c", Name: "C", Val: 3}, Action: "index"},
	}
	if err := client.BulkIndex(ctx, index, docs); err != nil {
		t.Fatalf("BulkIndex failed: %v", err)
	}
	var results []testDoc
	total, err := client.Search(ctx, index, nil, 1, 10, &results)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if total < 3 {
		t.Errorf("Expected at least 3 docs, got %d", total)
	}
	_ = client.Delete(ctx, index, "a")
	_ = client.Delete(ctx, index, "b")
	_ = client.Delete(ctx, index, "c")
	_ = os.Remove(testFilePath)
}

func TestLocalDeleteFileBased(t *testing.T) {
	_ = os.Remove(testFilePath)
	client := NewLocalDatastoreClient(testFilePath)
	ctx := context.Background()
	index := "testdelete"
	id := "del"
	doc := testDoc{ID: id, Name: "todel", Val: 99}
	if err := client.Index(ctx, index, id, doc); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	if err := client.Delete(ctx, index, id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	var got testDoc
	err := client.Get(ctx, index, id, &got)
	if err == nil {
		t.Errorf("Expected error for deleted doc, got: %+v", got)
	}
	_ = os.Remove(testFilePath)
}

func TestLocalPaginationFileBased(t *testing.T) {
	_ = os.Remove(testFilePath)
	client := NewLocalDatastoreClient(testFilePath)
	ctx := context.Background()
	index := "testpage"
	for i := 1; i <= 15; i++ {
		id := fmt.Sprintf("%d", i)
		doc := testDoc{ID: id, Name: id, Val: i}
		client.Index(ctx, index, id, doc)
	}
	var results []testDoc
	total, err := client.Search(ctx, index, nil, 2, 5, &results)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("Expected 5 docs on page 2, got %d", len(results))
	}
	if total < 15 {
		t.Errorf("Expected at least 15 docs, got %d", total)
	}
	// Cleanup
	for i := 1; i <= 15; i++ {
		id := fmt.Sprintf("%d", i)
		client.Delete(ctx, index, id)
	}
	_ = os.Remove(testFilePath)
}

func TestLocalScrollQueryFileBased(t *testing.T) {
	_ = os.Remove(testFilePath)
	client := NewLocalDatastoreClient(testFilePath)
	ctx := context.Background()
	index := "testscroll"
	// Insert 20 docs
	for i := 1; i <= 20; i++ {
		id := fmt.Sprintf("%d", i)
		doc := testDoc{ID: id, Name: "Name" + id, Val: i}
		client.Index(ctx, index, id, doc)
	}
	var allDocs []testDoc
	err := client.ScrollQuery(ctx, index, nil, 7, func(batch []json.RawMessage) bool {
		for _, raw := range batch {
			var doc testDoc
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Errorf("unmarshal error: %v", err)
				return false
			}
			allDocs = append(allDocs, doc)
		}
		return true
	})
	if err != nil {
		t.Fatalf("ScrollQuery failed: %v", err)
	}
	if len(allDocs) != 20 {
		t.Errorf("Expected 20 docs, got %d", len(allDocs))
	}
	// Cleanup
	for i := 1; i <= 20; i++ {
		id := fmt.Sprintf("%d", i)
		client.Delete(ctx, index, id)
	}
	_ = os.Remove(testFilePath)
}

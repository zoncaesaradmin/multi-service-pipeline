//go:build local
// +build local

package datastore

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

type localTestDoc struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestLocalIndexAndGet(t *testing.T) {
	client := NewLocalDatastoreClient()
	ctx := context.Background()
	index := "testindex"
	id := "1"
	doc := localTestDoc{ID: id, Name: "testname"}
	if err := client.Index(ctx, index, id, doc); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	var got localTestDoc
	if err := client.Get(ctx, index, id, &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != doc.Name {
		t.Errorf("Get returned wrong doc: %+v", got)
	}
	_ = client.Delete(ctx, index, id)
}

func TestLocalBulkIndexAndSearch(t *testing.T) {
	client := NewLocalDatastoreClient()
	ctx := context.Background()
	index := "testbulk"
	docs := []BulkDoc{
		{ID: "a", Body: localTestDoc{ID: "a", Name: "A"}, Action: "index"},
		{ID: "b", Body: localTestDoc{ID: "b", Name: "B"}, Action: "index"},
		{ID: "c", Body: localTestDoc{ID: "c", Name: "C"}, Action: "index"},
	}
	if err := client.BulkIndex(ctx, index, docs); err != nil {
		t.Fatalf("BulkIndex failed: %v", err)
	}
	var results []localTestDoc
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
}

func TestLocalDelete(t *testing.T) {
	client := NewLocalDatastoreClient()
	ctx := context.Background()
	index := "testdelete"
	id := "del"
	doc := localTestDoc{ID: id, Name: "todel"}
	if err := client.Index(ctx, index, id, doc); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	if err := client.Delete(ctx, index, id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	var got localTestDoc
	err := client.Get(ctx, index, id, &got)
	if err == nil {
		t.Errorf("Expected error for deleted doc, got: %+v", got)
	}
}

func TestLocalPagination(t *testing.T) {
	client := NewLocalDatastoreClient()
	ctx := context.Background()
	index := "testpage"
	for i := 1; i <= 15; i++ {
		id := string(rune('a' + i))
		doc := localTestDoc{ID: id, Name: id}
		client.Index(ctx, index, id, doc)
	}
	var results []localTestDoc
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
		id := string(rune('a' + i))
		client.Delete(ctx, index, id)
	}
}

func TestLocalScrollQuery(t *testing.T) {
	client := NewLocalDatastoreClient()
	ctx := context.Background()
	index := "testscroll"
	// Insert 20 docs
	for i := 1; i <= 20; i++ {
		id := fmt.Sprintf("%d", i)
		doc := localTestDoc{ID: id, Name: "Name" + id}
		client.Index(ctx, index, id, doc)
	}
	var allDocs []localTestDoc
	err := client.ScrollQuery(ctx, index, nil, 7, func(batch []json.RawMessage) bool {
		for _, raw := range batch {
			var doc localTestDoc
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
}

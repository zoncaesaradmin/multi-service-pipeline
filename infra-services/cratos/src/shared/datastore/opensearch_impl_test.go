package datastore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/disaster37/opensearch/v2"
)

var scrollBatchMap = make(map[string]int)

// mockOpenSearchServer returns a test server that simulates OpenSearch responses for all client methods
func mockOpenSearchServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Custom responses for each test case
		switch {
		case strings.HasPrefix(r.URL.Path, "/_bulk"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"errors":false}`))
		case strings.HasSuffix(r.URL.Path, "/_refresh"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"_shards":{"total":1,"successful":1,"failed":0}}`))
		case strings.Contains(r.URL.Path, "testindex/_doc/1") && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"created"}`))
		case strings.Contains(r.URL.Path, "testindex/_doc/1") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"found":true,"_source":{"id":"1","name":"testname"}}`))
		case strings.Contains(r.URL.Path, "testindex/_doc/1") && r.Method == "DELETE":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"deleted"}`))
		case strings.Contains(r.URL.Path, "testbulk/_search"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"hits":{"total":{"value":3},"hits":[{"_source":{"id":"a","name":"A"}},{"_source":{"id":"b","name":"B"}},{"_source":{"id":"c","name":"C"}}]}}`))
		case strings.Contains(r.URL.Path, "testdelete/_doc/del") && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"created"}`))
		case strings.Contains(r.URL.Path, "testdelete/_doc/del") && r.Method == "DELETE":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"deleted"}`))
		case strings.Contains(r.URL.Path, "testdelete/_doc/del") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"found":false}`))
		case strings.Contains(r.URL.Path, "testpage/_search"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"hits":{"total":{"value":15},"hits":[{"_source":{"id":"b","name":"b"}},{"_source":{"id":"c","name":"c"}},{"_source":{"id":"d","name":"d"}},{"_source":{"id":"e","name":"e"}},{"_source":{"id":"f","name":"f"}}]}}`))
		case strings.Contains(r.URL.Path, "testscroll/_search"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"_scroll_id":"dummy-scroll-id","hits":{"hits":[{"_source":{"id":"doc1","name":"Name1"}},{"_source":{"id":"doc2","name":"Name2"}},{"_source":{"id":"doc3","name":"Name3"}},{"_source":{"id":"doc4","name":"Name4"}},{"_source":{"id":"doc5","name":"Name5"}},{"_source":{"id":"doc6","name":"Name6"}},{"_source":{"id":"doc7","name":"Name7"}}]}}`))
		case strings.Contains(r.URL.Path, "_search/scroll"):
			// Parse scroll_id from request body if present
			scrollID := "dummy-scroll-id"
			batch := scrollBatchMap[scrollID]
			batch++
			scrollBatchMap[scrollID] = batch
			w.WriteHeader(http.StatusOK)
			if batch == 1 {
				w.Write([]byte(`{"_scroll_id":"dummy-scroll-id","hits":{"hits":[{"_source":{"id":"doc8","name":"Name8"}},{"_source":{"id":"doc9","name":"Name9"}},{"_source":{"id":"doc10","name":"Name10"}},{"_source":{"id":"doc11","name":"Name11"}},{"_source":{"id":"doc12","name":"Name12"}},{"_source":{"id":"doc13","name":"Name13"}},{"_source":{"id":"doc14","name":"Name14"}}]}}`))
			} else if batch == 2 {
				w.Write([]byte(`{"_scroll_id":"dummy-scroll-id","hits":{"hits":[{"_source":{"id":"doc15","name":"Name15"}},{"_source":{"id":"doc16","name":"Name16"}},{"_source":{"id":"doc17","name":"Name17"}},{"_source":{"id":"doc18","name":"Name18"}},{"_source":{"id":"doc19","name":"Name19"}},{"_source":{"id":"doc20","name":"Name20"}}]}}`))
			} else {
				w.Write([]byte(`{"_scroll_id":"dummy-scroll-id","hits":{"hits":[]},"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"took":1,"timed_out":false}`))
			}
		case strings.Contains(r.URL.Path, "_search/clear"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"succeeded":true,"num_freed":1}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"acknowledged":true}`))
		}
	}))
}

func getMockClient(t *testing.T) OpenSearchClient {
	server := mockOpenSearchServer()
	client, err := NewOpenSearchClient(
		opensearch.SetURL(server.URL),
		opensearch.SetHealthcheck(false),
		opensearch.SetSniff(false),
	)
	if err != nil {
		t.Fatalf("failed to create mock client: %v", err)
	}
	return client
}

func getTestClient(t *testing.T) OpenSearchClient {
	url := os.Getenv("OPENSEARCH_URL")
	if url == "" {
		url = "http://localhost:9200"
	}
	client, err := NewOpenSearchClient(opensearch.SetURL(url))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

type testDoc struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestIndexAndGet(t *testing.T) {
	client := getMockClient(t)
	ctx := context.Background()
	index := "testindex"
	id := "1"
	doc := testDoc{ID: id, Name: "testname"}
	if err := client.Index(ctx, index, id, doc); err != nil {
		t.Fatalf("Index failed: %v", err)
	}
	var got testDoc
	if err := client.Get(ctx, index, id, &got); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != doc.Name {
		t.Errorf("Get returned wrong doc: %+v", got)
	}
	_ = client.Delete(ctx, index, id)
}

func TestBulkIndexAndSearch(t *testing.T) {
	client := getMockClient(t)
	ctx := context.Background()
	index := "testbulk"
	docs := []BulkDoc{
		{ID: "a", Body: testDoc{ID: "a", Name: "A"}, Action: "index"},
		{ID: "b", Body: testDoc{ID: "b", Name: "B"}, Action: "index"},
		{ID: "c", Body: testDoc{ID: "c", Name: "C"}, Action: "index"},
	}
	if err := client.BulkIndex(ctx, index, docs); err != nil {
		t.Fatalf("BulkIndex failed: %v", err)
	}
	// Wait for index refresh
	client.(*OpenSearchClientImpl).client.Refresh(index).Do(ctx)
	var results []testDoc
	total, err := client.Search(ctx, index, map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}}, 1, 10, &results)
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

func TestDelete(t *testing.T) {
	client := getMockClient(t)
	ctx := context.Background()
	index := "testdelete"
	id := "del"
	doc := testDoc{ID: id, Name: "todel"}
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
}

func TestPagination(t *testing.T) {
	client := getMockClient(t)
	ctx := context.Background()
	index := "testpage"
	for i := 1; i <= 15; i++ {
		id := string(rune('a' + i))
		doc := testDoc{ID: id, Name: id}
		client.Index(ctx, index, id, doc)
	}
	client.(*OpenSearchClientImpl).client.Refresh(index).Do(ctx)
	var results []testDoc
	total, err := client.Search(ctx, index, map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}}, 2, 5, &results)
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

func TestScrollQuery(t *testing.T) {
	client := getMockClient(t)
	ctx := context.Background()
	index := "testscroll"
	// Insert 20 docs
	for i := 1; i <= 20; i++ {
		id := fmt.Sprintf("doc%d", i)
		doc := testDoc{ID: id, Name: fmt.Sprintf("Name%d", i)}
		client.Index(ctx, index, id, doc)
	}
	client.(*OpenSearchClientImpl).client.Refresh(index).Do(ctx)
	// Scroll query: match all, page size 7
	var allDocs []testDoc
	err := client.ScrollQuery(ctx, index, map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}}, 7, func(batch []json.RawMessage) bool {
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
		id := fmt.Sprintf("doc%d", i)
		client.Delete(ctx, index, id)
	}
}

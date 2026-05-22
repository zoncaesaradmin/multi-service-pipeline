//go:build !local
// +build !local

package datastore

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"platformkit/logging"
	"slices"
	"strings"
	"testing"
)

func newOpenSearchTestLogger(t *testing.T) logging.Logger {
	t.Helper()

	config := &logging.LoggerConfig{
		Level:         logging.InfoLevel,
		FilePath:      "/tmp/test_opensearch.log",
		LoggerName:    "test",
		ComponentName: "test",
		ServiceName:   "test",
	}
	logger, err := logging.NewLogger(config)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

func withOpenSearchURL(t *testing.T, url string) {
	t.Helper()
	original := os.Getenv("OPENSEARCH_URL")
	if err := os.Setenv("OPENSEARCH_URL", url); err != nil {
		t.Fatalf("failed to set OPENSEARCH_URL: %v", err)
	}
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("OPENSEARCH_URL")
			return
		}
		_ = os.Setenv("OPENSEARCH_URL", original)
	})
}

func newOpenSearchTestClient(t *testing.T, url string) *OpenSearchClient {
	t.Helper()
	withOpenSearchURL(t, url)

	client, err := NewOpenSearchClient(newOpenSearchTestLogger(t))
	if err != nil {
		t.Fatalf("NewOpenSearchClient failed: %v", err)
	}

	openSearchClient, ok := client.(*OpenSearchClient)
	if !ok {
		t.Fatalf("expected *OpenSearchClient, got %T", client)
	}
	return openSearchClient
}

func TestNewOpenSearchClientLoadsEnvConfig(t *testing.T) {
	client := newOpenSearchTestClient(t, "http://127.0.0.1:9200")

	if client.client == nil {
		t.Fatal("expected elastic client to be initialized")
	}
	if client.url != "http://127.0.0.1:9200" {
		t.Fatalf("expected primary URL to be loaded from env, got %q", client.url)
	}
	if client.isSecure {
		t.Fatal("expected http URL to be treated as non-secure")
	}
	if !slices.Equal(client.urls, []string{"http://127.0.0.1:9200"}) {
		t.Fatalf("unexpected configured URLs: %#v", client.urls)
	}
}

func TestUpsertIndexCreatesIndex(t *testing.T) {
	var (
		requestPath string
		requestBody string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		requestBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"acknowledged":true,"shards_acknowledged":true,"index":"alerts"}`))
	}))
	defer server.Close()

	client := newOpenSearchTestClient(t, server.URL)

	tempDir := t.TempDir()
	mappingPath := filepath.Join(tempDir, "alerts.json")
	mappingBody := `{"settings":{"number_of_shards":1}}`
	if err := os.WriteFile(mappingPath, []byte(mappingBody), 0o644); err != nil {
		t.Fatalf("failed to write mapping file: %v", err)
	}

	if err := client.UpsertIndex("alerts", mappingPath); err != nil {
		t.Fatalf("UpsertIndex failed: %v", err)
	}

	if requestPath != "/alerts" {
		t.Fatalf("expected create-index request to /alerts, got %q", requestPath)
	}
	if strings.TrimSpace(requestBody) != mappingBody {
		t.Fatalf("expected mapping body %q, got %q", mappingBody, requestBody)
	}
}

func TestExecuteQueryWithScrollCallbackProcessesAllBatches(t *testing.T) {
	scrollCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/_search/scroll"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"succeeded":true,"num_freed":1}`))
		case strings.Contains(r.URL.Path, "/events/_search"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"_scroll_id":"scroll-1",
				"hits":{"hits":[
					{"_id":"1","_source":{"name":"alpha"}},
					{"_id":"2","_source":{"name":"beta"}}
				]}
			}`))
		case strings.Contains(r.URL.Path, "/_search/scroll"):
			scrollCalls++
			w.WriteHeader(http.StatusOK)
			switch scrollCalls {
			case 1:
				_, _ = w.Write([]byte(`{
					"_scroll_id":"scroll-1",
					"hits":{"hits":[
						{"_id":"3","_source":{"name":"gamma"}}
					]}
				}`))
			default:
				_, _ = w.Write([]byte(`{"_scroll_id":"scroll-1","hits":{"hits":[]}}`))
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := newOpenSearchTestClient(t, server.URL)

	var batches [][]map[string]interface{}
	err := client.ExecuteQueryWithScrollCallback(
		"events",
		map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}},
		2,
		"1m",
		func(batch []map[string]interface{}) error {
			copied := make([]map[string]interface{}, len(batch))
			for i, doc := range batch {
				copied[i] = doc
			}
			batches = append(batches, copied)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ExecuteQueryWithScrollCallback failed: %v", err)
	}

	if len(batches) != 2 {
		t.Fatalf("expected 2 callback batches, got %d", len(batches))
	}
	if got := batches[0][0]["name"]; got != "alpha" {
		t.Fatalf("expected first document name alpha, got %#v", got)
	}
	if got := batches[0][0]["_id"]; got != "1" {
		t.Fatalf("expected first document _id 1, got %#v", got)
	}
	if got := batches[1][0]["name"]; got != "gamma" {
		t.Fatalf("expected second batch document name gamma, got %#v", got)
	}
}

func TestExecuteQueryWithScrollCallbackStopsOnErrStopScroll(t *testing.T) {
	scrollCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/_search/scroll"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"succeeded":true,"num_freed":1}`))
		case strings.Contains(r.URL.Path, "/events/_search"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"_scroll_id":"scroll-1",
				"hits":{"hits":[{"_id":"1","_source":{"name":"alpha"}}]}
			}`))
		case strings.Contains(r.URL.Path, "/_search/scroll"):
			scrollCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"_scroll_id":"scroll-1",
				"hits":{"hits":[{"_id":"2","_source":{"name":"beta"}}]}
			}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := newOpenSearchTestClient(t, server.URL)

	callbacks := 0
	err := client.ExecuteQueryWithScrollCallback(
		"events",
		map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}},
		1,
		"1m",
		func(batch []map[string]interface{}) error {
			callbacks++
			return ErrStopScroll
		},
	)
	if err != nil {
		t.Fatalf("expected ErrStopScroll to be treated as a clean stop, got %v", err)
	}
	if callbacks != 1 {
		t.Fatalf("expected callback to run once before stopping, got %d", callbacks)
	}
	if scrollCalls != 0 {
		t.Fatalf("expected no follow-up scroll request after ErrStopScroll, got %d", scrollCalls)
	}
}

func TestExecuteQueryWithScrollCallbackPropagatesJSONErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/_search/scroll"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"succeeded":true,"num_freed":1}`))
		case strings.Contains(r.URL.Path, "/events/_search"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"_scroll_id":"scroll-1",
				"hits":{"hits":[{"_id":"1","_source":"not-an-object"}]}
			}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := newOpenSearchTestClient(t, server.URL)

	err := client.ExecuteQueryWithScrollCallback(
		"events",
		map[string]interface{}{"query": map[string]interface{}{"match_all": map[string]interface{}{}}},
		1,
		"1m",
		func(batch []map[string]interface{}) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected invalid _source payload to return an error")
	}
}

func TestSplitURLs(t *testing.T) {
	urls := splitURLs("http://a:9200, http://b:9200\nhttp://c:9200")
	expected := []string{"http://a:9200", "http://b:9200", "http://c:9200"}
	if !slices.Equal(urls, expected) {
		t.Fatalf("expected URLs %#v, got %#v", expected, urls)
	}
}

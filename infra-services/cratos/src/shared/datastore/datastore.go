package datastore

import (
	context "context"
	"encoding/json"
)

// OpenSearchClient defines the interface for interacting with OpenSearch/Elasticsearch
// This interface is designed for optimal bulk reads, writes, and paginated queries.
type OpenSearchClient interface {
	// Index a single document
	Index(ctx context.Context, index string, id string, body interface{}) error

	// BulkIndex indexes multiple documents efficiently
	BulkIndex(ctx context.Context, index string, docs []BulkDoc) error

	// Get retrieves a document by ID
	Get(ctx context.Context, index string, id string, out interface{}) error

	// Search performs a query and returns results with pagination
	Search(ctx context.Context, index string, query interface{}, page, pageSize int, out interface{}) (total int, err error)

	// Delete removes a document by ID
	Delete(ctx context.Context, index string, id string) error

	// ScrollQuery executes a query with scroll and invokes callback for each batch of results
	ScrollQuery(ctx context.Context, index string, query interface{}, pageSize int, callback func([]json.RawMessage) bool) error

	// Close releases any resources held by the client
	Close() error
}

// BulkDoc represents a document for bulk operations
// ID can be empty for auto-generated IDs
// Body is the document content
// Action is typically "index", "create", "update", or "delete"
type BulkDoc struct {
	ID     string
	Body   interface{}
	Action string
}

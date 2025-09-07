package datastore

import (
	"context"
	"errors"
	"sharedgomodule/logging"
	"strings"
)

// ErrStopScroll is returned by the scroll callback to stop scrolling
var ErrStopScroll = errors.New("stop scroll")

// DatabaseClient defines the interface for datastore operations
type DatabaseClient interface {
	// Index management
	UpsertIndex(indexName string, mapFilePath string) error

	// Scroll operations
	ExecuteQueryWithScrollCallback(index string, query interface{}, scrollSize int, scrollTimeout string, process func(batch []map[string]interface{}) error) error
}

var EsIndices string = "activeanomalydb"

// EsTemplatePath defines the path to the configuration templates directory
var EsTemplatePath string = "/opt/cratos/conf"

// Consolidated client instance
var dbClient DatabaseClient

// GetDatabaseClient returns the singleton database client
func GetDatabaseClient() DatabaseClient { return dbClient }

// SetDatabaseClient allows injection from initialization code
func SetDatabaseClient(c DatabaseClient) { dbClient = c }

func Init(ctx context.Context, logger logging.Logger, indexList string) {
	// Initialize the consolidated client - using unified implementation
	dbClient = newDatabaseClient(logger)

	indices := strings.Split(indexList, ",")
	for _, ind := range indices {
		if len(ind) == 0 {
			continue
		}
		mappingPath := EsTemplatePath + "/" + ind + ".json"
		err := dbClient.UpsertIndex(EsIndexPrefix()+ind, mappingPath)
		if err != nil {
			logger.Errorw("datastore init - failed to update schema for index", "error", err, "index", ind)
		} else {
			logger.Infow("Successfully initialized index", "index", ind)
		}
	}
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

package ruletask

import (
	"context"
	"corekit/configutil"
	"corekit/datastore"
	"corekit/logging"
	"fmt"
	"os"
	"strings"
)

const (
	TaskStoreIndex = "alertruletask"

	// Task states
	TaskStatePending    = "pending"
	TaskStateProcessing = "processing"
	TaskStateCompleted  = "completed"
	TaskStateFailed     = "failed"
	TaskStateRetrying   = "retrying"

	// Error messages
	ErrDatabaseClientNotInitialized = "database client not initialized"
)

type TaskStore struct {
	client    datastore.DatabaseClient
	logger    logging.Logger
	hostname  string
	indexName string // Pre-calculated full index name with prefix
}

func NewTaskStore(logger logging.Logger) *TaskStore {
	hostname, _ := os.Hostname()
	dbClient := datastore.GetDatabaseClient()
	indexName := taskStoreIndexPrefix() + TaskStoreIndex

	if dbClient == nil {
		logger.Warnw("Database client is nil during TaskStore creation - this may cause issues")
	} else {
		logger.Debugw("TaskStore created with valid database client", "hostname", hostname, "index", indexName)
	}

	return &TaskStore{
		client:    dbClient,
		logger:    logger,
		hostname:  hostname,
		indexName: indexName,
	}
}

func taskStoreIndexPrefix() string {
	prefix := strings.TrimSpace(os.Getenv("ES_INDEX_PREFIX"))
	if prefix == "" {
		prefix = strings.TrimSpace(os.Getenv("OPENSEARCH_INDEX_PREFIX"))
	}
	if prefix == "" {
		cfg := configutil.LoadConfigMap(configutil.ResolveConfFilePath("opensearch.yaml"))
		if cfg != nil {
			if raw, ok := cfg["index_prefix"]; ok {
				if value, ok := raw.(string); ok {
					prefix = strings.TrimSpace(value)
				}
			}
		}
	}
	return prefix
}

func (ts *TaskStore) Initialize(ctx context.Context) error {
	if ts.client == nil {
		ts.logger.Errorw("TaskStore Initialize called but client is nil - database not initialized properly")
		return fmt.Errorf("%s", ErrDatabaseClientNotInitialized)
	}

	ts.logger.Infow("TaskStore initialized successfully",
		"index", ts.indexName,
		"hostname", ts.hostname)
	return nil
}

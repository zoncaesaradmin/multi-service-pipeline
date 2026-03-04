package ruletask

import (
	"context"
	"fmt"
	"os"
	"sharedgomodule/datastore"
	"sharedgomodule/logging"
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
	indexName := datastore.EsIndexPrefix() + TaskStoreIndex

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

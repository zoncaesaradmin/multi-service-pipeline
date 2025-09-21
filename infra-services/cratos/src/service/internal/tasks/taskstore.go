package tasks

import (
	"sharedgomodule/logging"
)

type TaskStore struct {
	logger    logging.LoggerConfig
	hostname  string
	indexName string // Pre-calculated full index name with prefix
}

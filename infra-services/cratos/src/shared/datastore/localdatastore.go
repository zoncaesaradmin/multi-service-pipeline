//go:build local
// +build local

package datastore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sharedgomodule/logging"
	"strings"
	"sync"
	"time"
)

// LocalClient implements DatabaseClient for local file-based storage
type LocalClient struct {
	logger  logging.Logger
	dataDir string
	mutex   sync.RWMutex
}

// NewLocalClient creates a new local file-based datastore client
func NewLocalClient(logger logging.Logger) DatabaseClient {
	dataDir := "/tmp/cratos-datastore-local"
	if customDir := os.Getenv("LOCAL_DATASTORE_DIR"); customDir != "" {
		dataDir = customDir
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Errorw("Failed to create local datastore directory", "error", err, "dir", dataDir)
	}
	return &LocalClient{
		logger:  logger,
		dataDir: dataDir,
	}
}

func (lc *LocalClient) UpsertIndex(indexName string, mapFilePath string) error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	indexDir := filepath.Join(lc.dataDir, indexName)
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory %s: %w", indexDir, err)
	}
	// Create a metadata file for the index
	metadataFile := filepath.Join(indexDir, "_metadata.json")
	metadata := map[string]interface{}{
		"index_name":   indexName,
		"mapping_file": mapFilePath,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}

	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index  metatdata: %w", err)
	}
	if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write index metadata: %w", err)
	}

	lc.logger.Infow("Local index created/updated", "index", indexName, "dir", indexDir)
	return nil
}

func (lc *LocalClient) ExecuteQueryWithScrollCallback(index string, query interface{}, scrollSize int, scrollTimeout string, process func(batch []map[string]interface{}) error) error {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	indexDir := filepath.Join(lc.dataDir, index)
	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		lc.logger.Warnw("Index directory does not exist", "index", index, "dir", indexDir)
		return nil
	}

	files, err := filepath.Glob(filepath.Join(indexDir, "doc_*.json"))
	if err != nil {
		return fmt.Errorf("failed to list documents in index %s: %w", index, err)
	}

	allDocs := lc.readDocuments(files)

	for start := 0; start < len(allDocs); start += scrollSize {
		end := start + scrollSize
		if end > len(allDocs) {
			end = len(allDocs)
		}
		batch := allDocs[start:end]
		if len(batch) == 0 {
			continue
		}
		err := process(batch)
		if err != nil {
			if err == ErrStopScroll {
				break
			}
			return err
		}
	}
	return nil
}

// readDocuments reads and parses documents from file paths, adding file-based IDs if missing.
func (lc *LocalClient) readDocuments(files []string) []map[string]interface{} {
	var docs []map[string]interface{}
	for _, file := range files {
		doc := lc.readDocument(file)
		if doc != nil {
			docs = append(docs, doc)
		}
	}
	return docs
}

// readDocument reads and parses a single document file, adding file-based ID if missing.
func (lc *LocalClient) readDocument(file string) map[string]interface{} {
	data, err := os.ReadFile(file)
	if err != nil {
		lc.logger.Warnw("Failed to read document file", "file", file, "error", err)
		return nil
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		lc.logger.Warnw("Failed to parse document JSON", "file", file, "error", err)
		return nil
	}
	if _, exists := doc["_id"]; !exists {
		fileName := filepath.Base(file)
		docId := strings.TrimPrefix(fileName, "doc_")
		docId = strings.TrimSuffix(docId, ".json")
		doc["_id"] = docId
	}
	return doc
}

func newDatabaseClient(logger logging.Logger) DatabaseClient {
	logger.Info("Initializing local file-based datastore client")
	return NewLocalClient(logger)
}

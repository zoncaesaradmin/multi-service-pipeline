//go:build local
// +build local

package datastore

import (
	"sharedgomodule/logging"
	"testing"
)

const (
	testLogFile       = "/tmp/test_datastore.log"
	testLoggerFailMsg = "Failed to create logger: %v"
	testIndex         = "test-index"
	testMappingFile   = "test-mapping.json"
)

const testFilePath = "test_localdatastore.json"

type testDoc struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
	Val  int    `json:"val"`
}

func createTestLogger() (logging.Logger, error) {
	config := &logging.LoggerConfig{
		Level:         logging.InfoLevel,
		FilePath:      testLogFile,
		LoggerName:    "test",
		ComponentName: "test",
		ServiceName:   "test",
	}
	return logging.NewLogger(config)
}

func TestLocalIndexAndGetFileBased(t *testing.T) {
	logger, err := createTestLogger()
	if err != nil {
		t.Fatalf(testLoggerFailMsg, err)
	}
	_ = NewLocalClient(logger)
}

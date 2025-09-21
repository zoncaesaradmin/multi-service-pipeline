package logging_test

import (
	"testing"

	"sharedgomodule/logging"
)

// TestMockLoggerUsage demonstrates how to use MockLogger in your tests
func TestMockLoggerUsage(t *testing.T) {
	// Create a mock logger
	mockLogger := logging.NewMockLogger()
	mockLogger.SetComponentName("example-service")
	mockLogger.SetServiceName("test")

	// Use the logger in your code under test
	service := &ExampleService{logger: mockLogger}
	service.ProcessMessage("hello world")

	// Debug: Print actual log entries
	entries := mockLogger.GetLogEntries()
	for i, entry := range entries {
		t.Logf("Entry %d: Level=%s, Message=%q, Fields=%+v", i, entry.Level.String(), entry.Message, entry.Fields)
	}

	// The service logs "Processing message: hello world" with Infof
	if !mockLogger.HasLogEntryContaining(logging.InfoLevel, "Processing message") {
		t.Error("Expected processing log message")
		return
	}

	if !mockLogger.HasLogEntryWithField(logging.InfoLevel, "component", "example-service") {
		t.Error("Expected component field")
		return
	}
}

// ExampleService demonstrates a service that uses logging
type ExampleService struct {
	logger logging.Logger
}

func (s *ExampleService) ProcessMessage(msg string) {
	// All logs go to the same logger instance for testing simplicity
	s.logger.Infof("Processing message: %s", msg)
	s.logger.Infof("Message length: %d", len(msg))
	s.logger.Info("Message processed successfully")
}

// Example test showing how to replace an existing mock implementation
func TestServiceWithMockLogger(t *testing.T) {
	// Instead of creating a local mock, use the shared one
	mockLogger := logging.NewMockLoggerWithLevel(logging.DebugLevel)
	mockLogger.SetComponentName("user-service")

	// Create service with mock logger
	service := &ExampleService{logger: mockLogger}

	// Test the service
	service.ProcessMessage("test message")

	// Verify behavior using rich mock methods
	if mockLogger.GetLogCount() != 3 {
		t.Errorf("Expected 3 log entries, got %d", mockLogger.GetLogCount())
	}

	// Verify specific log content
	if !mockLogger.HasLogEntryContaining(logging.InfoLevel, "Processing message") {
		t.Error("Expected processing log message")
	}

	// Verify message length was logged
	if !mockLogger.HasLogEntryContaining(logging.InfoLevel, "Message length: 12") {
		t.Error("Expected message length log")
	}

	// Verify component was added to all logs
	entries := mockLogger.GetLogEntries()
	for _, entry := range entries {
		if entry.Fields["component"] != "user-service" {
			t.Errorf("Expected component field in all entries, missing in: %+v", entry)
		}
	}
}

// Example showing how to test error scenarios
func TestServiceErrorHandling(t *testing.T) {
	mockLogger := logging.NewMockLogger()

	service := &ErrorService{logger: mockLogger}
	service.FailingOperation()

	// Verify error was logged
	if !mockLogger.HasLogEntryContaining(logging.ErrorLevel, "Operation failed") {
		t.Error("Expected error log message")
	}

	// Verify error count
	if mockLogger.GetLogCountByLevel(logging.ErrorLevel) != 1 {
		t.Error("Expected exactly 1 error log")
	}
}

type ErrorService struct {
	logger logging.Logger
}

func (s *ErrorService) FailingOperation() {
	s.logger.Error("Operation failed with critical error")
}

// Example showing panic behavior testing
func TestServicePanicLogging(t *testing.T) {
	mockLogger := logging.NewMockLogger()
	// Enable panic on fatal for testing fatal error paths
	mockLogger.EnablePanicOnFatal()

	service := &PanicService{logger: mockLogger}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from fatal log")
		}

		// Verify the fatal log was captured before panic
		if !mockLogger.HasLogEntry(logging.FatalLevel, "Critical system failure") {
			t.Error("Expected fatal log entry")
		}
	}()

	service.CriticalFailure()
}

type PanicService struct {
	logger logging.Logger
}

func (s *PanicService) CriticalFailure() {
	s.logger.Fatal("Critical system failure")
}

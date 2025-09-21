package logging

import (
	"context"
	"errors"
	"testing"
)

func TestMockLogger_BasicLogging(t *testing.T) {
	logger := NewMockLogger()
	logger.SetLevel(DebugLevel) // Set to debug level to capture all messages

	// Test basic logging methods
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	entries := logger.GetLogEntries()
	if len(entries) != 4 {
		t.Errorf("Expected 4 log entries, got %d", len(entries))
	}

	// Verify each entry
	expectedLevels := []Level{DebugLevel, InfoLevel, WarnLevel, ErrorLevel}
	expectedMessages := []string{"debug message", "info message", "warn message", "error message"}

	for i, entry := range entries {
		if entry.Level != expectedLevels[i] {
			t.Errorf("Expected level %v, got %v", expectedLevels[i], entry.Level)
		}
		if entry.Message != expectedMessages[i] {
			t.Errorf("Expected message %q, got %q", expectedMessages[i], entry.Message)
		}
	}
}

func TestMockLogger_FormattedLogging(t *testing.T) {
	logger := NewMockLogger()

	logger.Infof("Hello %s, age %d", "World", 42)
	logger.Errorf("Error code: %d - %s", 500, "Internal Server Error")

	entries := logger.GetLogEntries()
	if len(entries) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(entries))
	}

	if entries[0].Message != "Hello World, age 42" {
		t.Errorf("Expected formatted message, got %q", entries[0].Message)
	}

	if entries[1].Message != "Error code: 500 - Internal Server Error" {
		t.Errorf("Expected formatted error message, got %q", entries[1].Message)
	}
}

func TestMockLogger_VariadicLogging(t *testing.T) {
	logger := NewMockLogger()

	logger.Infow("test message", "key1", "value1", "key2", 42, "key3", true)

	entries := logger.GetLogEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Fields["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", entry.Fields["key1"])
	}
	if entry.Fields["key2"] != 42 {
		t.Errorf("Expected key2=42, got %v", entry.Fields["key2"])
	}
	if entry.Fields["key3"] != true {
		t.Errorf("Expected key3=true, got %v", entry.Fields["key3"])
	}
}

func TestMockLogger_LevelControl(t *testing.T) {
	logger := NewMockLogger()

	// Test default level
	if logger.GetLevel() != InfoLevel {
		t.Errorf("Expected default level InfoLevel, got %v", logger.GetLevel())
	}

	// Set to warn level
	logger.SetLevel(WarnLevel)

	// These should be ignored
	logger.Debug("debug")
	logger.Info("info")

	// These should be logged
	logger.Warn("warn")
	logger.Error("error")

	entries := logger.GetLogEntries()
	if len(entries) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(entries))
	}

	if !logger.IsLevelEnabled(WarnLevel) {
		t.Error("WarnLevel should be enabled")
	}
	if logger.IsLevelEnabled(InfoLevel) {
		t.Error("InfoLevel should not be enabled")
	}
	if !logger.IsLevelEnabled(ErrorLevel) {
		t.Error("ErrorLevel should be enabled")
	}
}

func TestMockLogger_WithFields(t *testing.T) {
	logger := NewMockLogger()

	fieldLogger := logger.WithFields(Fields{
		"component": "test",
		"version":   "1.0",
		"userId":    12345,
	})

	fieldLogger.Info("test message")

	// Original logger should not have the message
	if len(logger.GetLogEntries()) != 0 {
		t.Error("Original logger should not have log entries")
	}

	// Field logger should have the message with fields
	mockFieldLogger := fieldLogger.(*MockLogger)
	entries := mockFieldLogger.GetLogEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Fields["component"] != "test" {
		t.Errorf("Expected component=test, got %v", entry.Fields["component"])
	}
	if entry.Fields["version"] != "1.0" {
		t.Errorf("Expected version=1.0, got %v", entry.Fields["version"])
	}
	if entry.Fields["userId"] != 12345 {
		t.Errorf("Expected userId=12345, got %v", entry.Fields["userId"])
	}
}

func TestMockLogger_WithField(t *testing.T) {
	logger := NewMockLogger()

	fieldLogger := logger.WithField("requestId", "req-123")
	fieldLogger.Info("processing request")

	mockFieldLogger := fieldLogger.(*MockLogger)
	entries := mockFieldLogger.GetLogEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(entries))
	}

	if entries[0].Fields["requestId"] != "req-123" {
		t.Errorf("Expected requestId=req-123, got %v", entries[0].Fields["requestId"])
	}
}

func TestMockLogger_WithError(t *testing.T) {
	logger := NewMockLogger()
	testErr := errors.New("database connection failed")

	errorLogger := logger.WithError(testErr)
	errorLogger.Error("operation failed")

	mockErrorLogger := errorLogger.(*MockLogger)
	entries := mockErrorLogger.GetLogEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Fields["error"] != "database connection failed" {
		t.Errorf("Expected error field, got %v", entry.Fields["error"])
	}

	// Test with nil error
	nilErrorLogger := logger.WithError(nil)
	if nilErrorLogger != logger {
		t.Error("WithError(nil) should return the same logger")
	}
}

func TestMockLogger_WithContext(t *testing.T) {
	logger := NewMockLogger()

	ctx := context.WithValue(context.Background(), "traceID", "trace-123")
	ctx = context.WithValue(ctx, "userID", "user-456")
	ctx = context.WithValue(ctx, "requestID", "req-789")

	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("context message")

	mockCtxLogger := ctxLogger.(*MockLogger)
	entries := mockCtxLogger.GetLogEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Fields["traceID"] != "trace-123" {
		t.Errorf("Expected traceID=trace-123, got %v", entry.Fields["traceID"])
	}
	if entry.Fields["userID"] != "user-456" {
		t.Errorf("Expected userID=user-456, got %v", entry.Fields["userID"])
	}
	if entry.Fields["requestID"] != "req-789" {
		t.Errorf("Expected requestID=req-789, got %v", entry.Fields["requestID"])
	}
}

func TestMockLogger_Clone(t *testing.T) {
	logger := NewMockLogger()
	logger.SetLevel(ErrorLevel)
	logger.SetComponentName("original-component")
	logger.SetServiceName("original-service")
	logger = logger.WithField("persistent", "value").(*MockLogger)

	cloned := logger.Clone().(*MockLogger)

	// Test that clone has same configuration
	if cloned.GetLevel() != ErrorLevel {
		t.Error("Cloned logger should have same level")
	}
	if cloned.GetComponentName() != "original-component" {
		t.Error("Cloned logger should have same component name")
	}
	if cloned.GetServiceName() != "original-service" {
		t.Error("Cloned logger should have same service name")
	}

	// Test that logging to clone doesn't affect original
	cloned.Error("cloned message")

	if len(logger.GetLogEntries()) != 0 {
		t.Error("Original logger should not have log entries from clone")
	}

	if len(cloned.GetLogEntries()) != 1 {
		t.Error("Cloned logger should have its own log entries")
	}
}

func TestMockLogger_HelperMethods(t *testing.T) {
	logger := NewMockLogger()

	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	logger.Info("another info message")

	// Test GetLogCount
	if logger.GetLogCount() != 4 {
		t.Errorf("Expected 4 total logs, got %d", logger.GetLogCount())
	}

	// Test GetLogCountByLevel
	if logger.GetLogCountByLevel(InfoLevel) != 2 {
		t.Errorf("Expected 2 info logs, got %d", logger.GetLogCountByLevel(InfoLevel))
	}

	// Test HasLogEntry
	if !logger.HasLogEntry(WarnLevel, "warn message") {
		t.Error("Should find exact warn message")
	}
	if logger.HasLogEntry(WarnLevel, "nonexistent message") {
		t.Error("Should not find nonexistent message")
	}

	// Test HasLogEntryContaining
	if !logger.HasLogEntryContaining(InfoLevel, "another") {
		t.Error("Should find message containing 'another'")
	}
	if logger.HasLogEntryContaining(InfoLevel, "nonexistent") {
		t.Error("Should not find message containing 'nonexistent'")
	}

	// Test GetLogEntryWithMessage
	entry := logger.GetLogEntryWithMessage(ErrorLevel, "error message")
	if entry == nil {
		t.Error("Should find error log entry")
	}
	if entry != nil && entry.Message != "error message" {
		t.Errorf("Expected 'error message', got %q", entry.Message)
	}

	// Test GetLogEntriesContaining
	containing := logger.GetLogEntriesContaining(InfoLevel, "info")
	if len(containing) != 2 {
		t.Errorf("Expected 2 entries containing 'info', got %d", len(containing))
	}

	// Test ClearLogEntries
	logger.ClearLogEntries()
	if logger.GetLogCount() != 0 {
		t.Error("Expected 0 logs after clear")
	}
}

func TestMockLogger_ComponentAndService(t *testing.T) {
	logger := NewMockLogger()
	logger.SetComponentName("test-component")
	logger.SetServiceName("test-service")

	logger.Info("test message")

	entries := logger.GetLogEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Fields["component"] != "test-component" {
		t.Errorf("Expected component=test-component, got %v", entry.Fields["component"])
	}
	if entry.Fields["service"] != "test-service" {
		t.Errorf("Expected service=test-service, got %v", entry.Fields["service"])
	}

	// Test getters
	if logger.GetComponentName() != "test-component" {
		t.Errorf("Expected component name 'test-component', got %q", logger.GetComponentName())
	}
	if logger.GetServiceName() != "test-service" {
		t.Errorf("Expected service name 'test-service', got %q", logger.GetServiceName())
	}
}

func TestMockLogger_PanicAndFatal(t *testing.T) {
	logger := NewMockLogger()

	// Test without panic/fatal behavior (default)
	logger.Fatal("fatal message")
	logger.Panic("panic message")

	if logger.GetLogCount() != 2 {
		t.Errorf("Expected 2 log entries, got %d", logger.GetLogCount())
	}

	// Test panic behavior
	logger.EnablePanicOnPanic()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when ShouldPanic is true")
		}
	}()
	logger.Panic("should panic")
}

func TestMockLogger_FatalPanic(t *testing.T) {
	logger := NewMockLogger()
	logger.EnablePanicOnFatal()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when ShouldFatal is true")
		}
	}()
	logger.Fatal("should cause panic")
}

func TestMockLogger_LogMethods(t *testing.T) {
	logger := NewMockLogger()

	// Test Log method
	logger.Log(InfoLevel, "log message")

	// Test Logf method
	logger.Logf(WarnLevel, "formatted %s %d", "message", 42)

	// Test Logw method
	logger.Logw(ErrorLevel, "variadic message", "key1", "value1", "key2", 123)

	entries := logger.GetLogEntries()
	if len(entries) != 3 {
		t.Errorf("Expected 3 log entries, got %d", len(entries))
	}

	if entries[0].Level != InfoLevel || entries[0].Message != "log message" {
		t.Error("First entry should be info level with 'log message'")
	}

	if entries[1].Level != WarnLevel || entries[1].Message != "formatted message 42" {
		t.Errorf("Second entry should be warn level with formatted message, got %q", entries[1].Message)
	}

	if entries[2].Level != ErrorLevel || entries[2].Fields["key1"] != "value1" || entries[2].Fields["key2"] != 123 {
		t.Error("Third entry should be error level with key=value fields")
	}
}

func TestMockLogger_WithConfig(t *testing.T) {
	config := &LoggerConfig{
		Level:         WarnLevel,
		ComponentName: "config-component",
		ServiceName:   "config-service",
	}

	logger := NewMockLoggerWithConfig(config)

	if logger.GetLevel() != WarnLevel {
		t.Errorf("Expected level %v, got %v", WarnLevel, logger.GetLevel())
	}
	if logger.GetComponentName() != "config-component" {
		t.Errorf("Expected component 'config-component', got %q", logger.GetComponentName())
	}
	if logger.GetServiceName() != "config-service" {
		t.Errorf("Expected service 'config-service', got %q", logger.GetServiceName())
	}

	// Test with nil config
	nilLogger := NewMockLoggerWithConfig(nil)
	if nilLogger.GetLevel() != InfoLevel {
		t.Error("Should have default InfoLevel with nil config")
	}
}

func TestMockLogger_HasLogEntryWithFields(t *testing.T) {
	logger := NewMockLogger()

	// Create a logger with fields and log a message
	fieldLogger := logger.WithFields(Fields{
		"component": "test",
		"action":    "create",
		"id":        123,
	})
	fieldLogger.Info("test message")

	// The field logger should have the log entry with fields
	mockFieldLogger := fieldLogger.(*MockLogger)

	// Test finding entry with exact fields
	if !mockFieldLogger.HasLogEntryWithFields(InfoLevel, Fields{
		"component": "test",
		"action":    "create",
	}) {
		t.Error("Should find entry with matching fields")
	}

	// Test not finding entry with non-matching fields
	if mockFieldLogger.HasLogEntryWithFields(InfoLevel, Fields{
		"component": "test",
		"action":    "delete", // wrong value
	}) {
		t.Error("Should not find entry with non-matching field value")
	}

	// Test not finding entry with missing fields
	if mockFieldLogger.HasLogEntryWithFields(InfoLevel, Fields{
		"component":   "test",
		"nonexistent": "field",
	}) {
		t.Error("Should not find entry with non-existent field")
	}
}

func TestMockLogger_Close(t *testing.T) {
	logger := NewMockLogger()
	logger.Info("test message")

	if logger.GetLogCount() != 1 {
		t.Error("Should have 1 log entry before close")
	}

	err := logger.Close()
	if err != nil {
		t.Errorf("Close should not return error, got %v", err)
	}

	// After close, log entries should be cleared
	if logger.GetLogCount() != 0 {
		t.Error("Should have 0 log entries after close")
	}
}

func TestMockLogger_String(t *testing.T) {
	logger := NewMockLogger()
	logger.SetLevel(WarnLevel)
	logger.SetComponentName("test-comp")
	logger.SetServiceName("test-svc")
	logger.Info("test") // This won't be logged due to level
	logger.Warn("warn") // This will be logged

	str := logger.String()
	expected := "MockLogger{level:WARN, entries:1, component:test-comp, service:test-svc}"
	if str != expected {
		t.Errorf("Expected %q, got %q", expected, str)
	}
}

func TestMockLogger_PanicBehaviorControl(t *testing.T) {
	logger := NewMockLogger()

	// Test enabling and disabling panic behavior
	logger.EnablePanicOnPanic()
	logger.EnablePanicOnFatal()

	// Disable and test that no panic occurs
	logger.DisablePanicBehavior()

	logger.Fatal("should not panic")
	logger.Panic("should not panic")

	if logger.GetLogCount() != 2 {
		t.Errorf("Expected 2 log entries, got %d", logger.GetLogCount())
	}
}

func TestMockLogger_NewMockLoggerWithLevel(t *testing.T) {
	logger := NewMockLoggerWithLevel(ErrorLevel)

	if logger.GetLevel() != ErrorLevel {
		t.Errorf("Expected ErrorLevel, got %v", logger.GetLevel())
	}

	// Test that only error and above are logged
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	if logger.GetLogCount() != 1 {
		t.Errorf("Expected 1 log entry, got %d", logger.GetLogCount())
	}

	entries := logger.GetLogEntries()
	if len(entries) > 0 && entries[0].Level != ErrorLevel {
		t.Error("Only error level should be logged")
	}
}

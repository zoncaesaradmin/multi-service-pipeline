package logging

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// ContextKey is a type for context keys to avoid collisions
type contextKey string

const (
	traceIDKey contextKey = "trace_id"

	debugMessage = "Debug message"
)

func TestNewLoggerWithConfig(t *testing.T) {
	logFile := "/tmp/test_new_logger.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         DebugLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test basic functionality
	if logger.GetLevel() != DebugLevel {
		t.Errorf("GetLevel() = %v, want %v", logger.GetLevel(), DebugLevel)
	}

	// Test logging
	logger.Info("Test message")
	logger.Debug(debugMessage)
	logger.Warn("Warning message")
	logger.Error("Error message")

	// Check if log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf(logFileNotCreatedFmt, err)
	}

	os.Remove(logFile)
}

func TestNewLoggerWithConfigFileError(t *testing.T) {
	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      "/invalid/path/test.log",
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err == nil {
		if logger != nil {
			logger.Close()
		}
		t.Error("NewLoggerWithConfig() expected error for invalid file path but got none")
	}
}

func TestZerologLoggerSetLevel(t *testing.T) {
	logFile := "/tmp/test_set_level.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test initial level
	if logger.GetLevel() != InfoLevel {
		t.Errorf("Initial level = %v, want %v", logger.GetLevel(), InfoLevel)
	}

	// Test setting level
	logger.SetLevel(ErrorLevel)
	if logger.GetLevel() != ErrorLevel {
		t.Errorf("After SetLevel(ErrorLevel) = %v, want %v", logger.GetLevel(), ErrorLevel)
	}

	// Test level filtering after change
	if logger.IsLevelEnabled(InfoLevel) {
		t.Error("Info level should not be enabled after setting level to Error")
	}

	if !logger.IsLevelEnabled(ErrorLevel) {
		t.Error("Error level should be enabled after setting level to Error")
	}

	os.Remove(logFile)
}

func TestZerologLoggerFormattedLogging(t *testing.T) {
	logFile := "/tmp/test_formatted_logging.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         DebugLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test formatted logging methods
	logger.Debugf("Debug message with %s and %d", "string", 42)
	logger.Infof("Info message with %s", "parameter")
	logger.Warnf("Warning message with %d warnings", 3)
	logger.Errorf("Error message with error code %d", 500)

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

func TestZerologLoggerVariadicLogging(t *testing.T) {
	logFile := "/tmp/test_variadic_logging.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         DebugLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test variadic logging methods
	logger.Debugw(debugMessage, "key1", "value1", "key2", 42)
	logger.Infow("Info message", "user", "john", "action", "login")
	logger.Warnw("Warning message", "count", 3, "threshold", 5)
	logger.Errorw("Error message", "error_code", 500, "message", "internal error")

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

func TestZerologLoggerWithFields(t *testing.T) {
	logFile := "/tmp/test_with_fields.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test WithFields
	fieldLogger := logger.WithFields(Fields{
		"user_id":    123,
		"session_id": "abc-def-ghi",
		"module":     "auth",
	})

	fieldLogger.Info("User logged in")
	fieldLogger.Warn("Password attempt failed")

	// Test WithField
	singleFieldLogger := logger.WithField("request_id", "req-12345")
	singleFieldLogger.Info("Processing request")

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

func TestZerologLoggerWithError(t *testing.T) {
	logFile := "/tmp/test_with_error.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test WithError with actual error
	testErr := errors.New("test error message")
	errorLogger := logger.WithError(testErr)
	errorLogger.Error("An error occurred")

	// Test WithError with nil error (should not add error field)
	nilErrorLogger := logger.WithError(nil)
	nilErrorLogger.Info("No error here")

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

func TestZerologLoggerWithContext(t *testing.T) {
	logFile := "/tmp/test_with_context.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test WithContext
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-123")
	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("Request processed")

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

func TestZerologLoggerGenericLogging(t *testing.T) {
	logFile := "/tmp/test_generic_logging.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         DebugLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test Log, Logf, and Logw methods
	logger.Log(InfoLevel, "Generic info message")
	logger.Logf(WarnLevel, "Generic warning with %d items", 5)
	logger.Logw(ErrorLevel, "Generic error", "code", 404, "path", "/api/users")

	// Test with disabled level
	logger.Log(Level(999), "This should not log")

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

func TestZerologLoggerClone(t *testing.T) {
	logFile := "/tmp/test_clone.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Add some fields to original logger
	originalWithFields := logger.WithFields(Fields{"original": "value", "shared": "original"})

	// Clone the logger
	clonedLogger := originalWithFields.Clone()

	// Add different fields to cloned logger
	clonedWithFields := clonedLogger.WithFields(Fields{"cloned": "value", "shared": "cloned"})

	// Log with both loggers
	originalWithFields.Info("Message from original logger")
	clonedWithFields.Info("Message from cloned logger")

	// Verify both can log independently
	if originalWithFields.GetLevel() != clonedLogger.GetLevel() {
		t.Error("Cloned logger should have same level as original")
	}

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

func TestZerologLoggerLevelConversion(t *testing.T) {
	tests := []struct {
		ourLevel     Level
		zerologLevel string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
		{PanicLevel, "panic"},
		{Level(999), "info"}, // Unknown level defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.ourLevel.String(), func(t *testing.T) {
			logFile := "/tmp/test_level_conversion.log"
			os.Remove(logFile)

			config := &LoggerConfig{
				Level:         tt.ourLevel,
				FilePath:      logFile,
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			}

			logger, err := NewLoggerWithConfig(config)
			if err != nil {
				t.Fatalf(newLoggerErrorFmt, err)
			}
			defer logger.Close()

			// Just verify the logger was created successfully with the level
			if logger.GetLevel() != tt.ourLevel {
				t.Errorf("Logger level = %v, want %v", logger.GetLevel(), tt.ourLevel)
			}

			os.Remove(logFile)
		})
	}
}

func TestZerologLoggerClose(t *testing.T) {
	logFile := "/tmp/test_close.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}

	// Log something first
	logger.Info("Test message before close")

	// Close the logger
	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Try to close again (should not error)
	err = logger.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}

	os.Remove(logFile)
}

func TestZerologLoggerLevelFiltering(t *testing.T) {
	logFile := "/tmp/test_level_filtering_detailed.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         WarnLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test that debug and info are filtered out
	logger.Debug("This debug message should not appear")
	logger.Info("This info message should not appear")
	logger.Debugf("This debug formatted message should not appear: %s", "test")
	logger.Infof("This info formatted message should not appear: %s", "test")

	// Test that warn and error go through
	logger.Warn("This warning should appear")
	logger.Error("This error should appear")
	logger.Warnf("This warning formatted message should appear: %s", "test")
	logger.Errorf("This error formatted message should appear: %s", "test")

	// Read the log file and verify only warn/error messages are present
	if content, err := os.ReadFile(logFile); err != nil {
		t.Errorf("Failed to read log file: %v", err)
	} else {
		contentStr := string(content)
		if strings.Contains(contentStr, "debug") || strings.Contains(contentStr, "info") {
			// This might contain the level field, so check for the actual message content
			if strings.Contains(contentStr, "This debug message") || strings.Contains(contentStr, "This info message") {
				t.Error("Debug/Info messages should be filtered out but were found in log file")
			}
		}
		if !strings.Contains(contentStr, "warning") || !strings.Contains(contentStr, "error") {
			t.Error("Warning/Error messages should be present in log file")
		}
	}

	os.Remove(logFile)
}

// TestZerologLoggerWarnErrorEdgeCases tests edge cases for Warn and Error methods to improve coverage
func TestZerologLoggerWarnErrorEdgeCases(t *testing.T) {
	logFile := "/tmp/test_warn_error_edge_cases.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         DebugLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test when level is above warn/error (should still log)
	logger.SetLevel(WarnLevel)
	logger.Warn("Warning at warn level")
	logger.Error("Error at warn level")

	// Test with fields added before warn/error
	fieldLogger := logger.WithField("request_id", "123")
	fieldLogger.Warn("Warning with field")
	fieldLogger.Error("Error with field")

	// Test Warnf and Errorf with different parameter types
	logger.Warnf("Warning with int %d, string %s, bool %t", 42, "test", true)
	logger.Errorf("Error with float %.2f, complex %v", 3.14, complex(1, 2))

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

// TestZerologLoggerLogMethodsEdgeCases tests edge cases for Log and Logf methods to improve coverage
func TestZerologLoggerLogMethodsEdgeCases(t *testing.T) {
	logFile := "/tmp/test_log_methods_edge_cases.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         DebugLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test Log method with all valid levels
	logger.Log(DebugLevel, "Debug via Log method")
	logger.Log(InfoLevel, "Info via Log method")
	logger.Log(WarnLevel, "Warn via Log method")
	logger.Log(ErrorLevel, "Error via Log method")

	// Test Logf method with all valid levels
	logger.Logf(DebugLevel, "Debug via Logf method: %s", "param")
	logger.Logf(InfoLevel, "Info via Logf method: %d", 123)
	logger.Logf(WarnLevel, "Warn via Logf method: %v", true)
	logger.Logf(ErrorLevel, "Error via Logf method: %f", 3.14)

	// Test with unknown level (should use info level)
	unknownLevel := Level(999)
	logger.Log(unknownLevel, "Message with unknown level")
	logger.Logf(unknownLevel, "Formatted message with unknown level: %s", "test")

	// Test when logger level is set higher (some should be filtered)
	logger.SetLevel(WarnLevel)
	logger.Log(DebugLevel, "Debug should be filtered")
	logger.Log(InfoLevel, "Info should be filtered")
	logger.Log(WarnLevel, "Warn should appear")
	logger.Log(ErrorLevel, "Error should appear")

	logger.Logf(DebugLevel, "Debug formatted should be filtered: %s", "test")
	logger.Logf(InfoLevel, "Info formatted should be filtered: %s", "test")
	logger.Logf(WarnLevel, "Warn formatted should appear: %s", "test")
	logger.Logf(ErrorLevel, "Error formatted should appear: %s", "test")

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

// TestZerologLoggerUnknownLevelHandling tests getEvent method with unknown levels for full coverage
func TestZerologLoggerUnknownLevelHandling(t *testing.T) {
	logFile := "/tmp/test_unknown_level_handling.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         DebugLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test various unknown levels to trigger default case in getEvent
	unknownLevels := []Level{
		Level(-1),  // Negative level
		Level(100), // High level
		Level(999), // Very high level
		Level(7),   // Between valid levels
	}

	for i, level := range unknownLevels {
		// Test basic logging methods with unknown levels
		logger.Log(level, "Unknown level message")
		logger.Logf(level, "Unknown level formatted message %d: %s", i, "test")
		logger.Logw(level, "Unknown level variadic message", "index", i, "level", int(level))
	}

	// Test all logging methods with an unknown level to ensure getEvent default case is hit
	unknownLevel := Level(999)
	logger.Debug(debugMessage)                       // This will use getEvent with DebugLevel
	logger.Info("Info message")                      // This will use getEvent with InfoLevel
	logger.Log(unknownLevel, "Direct unknown level") // This will use getEvent with unknown level

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

// TestZerologLoggerComplexFieldCombinations tests complex combinations of fields, errors, and context
func TestZerologLoggerComplexFieldCombinations(t *testing.T) {
	logFile := "/tmp/test_complex_combinations.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test complex chaining
	testErr := errors.New("test error")
	ctx := context.WithValue(context.Background(), traceIDKey, "trace-123")

	complexLogger := logger.
		WithContext(ctx).
		WithError(testErr).
		WithFields(Fields{"module": "test", "operation": "complex"}).
		WithField("user_id", 456)

	complexLogger.Info("Complex logger message")
	complexLogger.Warn("Complex logger warning")
	complexLogger.Error("Complex logger error")

	// Test method chaining in different orders
	logger.WithField("order", "first").WithError(testErr).Info("Chained first")
	logger.WithError(testErr).WithField("order", "second").Info("Chained second")
	logger.WithFields(Fields{"multiple": "fields"}).WithContext(ctx).Info("Chained third")

	// Test with nil values
	logger.WithError(nil).WithContext(context.Background()).Info("With nil values")

	// Check if log file was created and has content
	if stat, err := os.Stat(logFile); err != nil {
		t.Errorf(logFileNotCreatedFmt, err)
	} else if stat.Size() == 0 {
		t.Error(logFileEmptyMsg)
	}

	os.Remove(logFile)
}

// TestZerologLoggerLevelBoundaryTesting tests exact level boundary conditions
func TestZerologLoggerLevelBoundaryTesting(t *testing.T) {
	logFile := "/tmp/test_level_boundary.log"
	os.Remove(logFile)

	config := &LoggerConfig{
		Level:         WarnLevel, // Set to warn level
		FilePath:      logFile,
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLoggerWithConfig(config)
	if err != nil {
		t.Fatalf(newLoggerErrorFmt, err)
	}
	defer logger.Close()

	// Test exact boundary: Warn level logger
	// These should be filtered (below warn level)
	logger.Debug("Debug should be filtered")
	logger.Info("Info should be filtered")
	logger.Debugf("Debug formatted should be filtered")
	logger.Infof("Info formatted should be filtered")
	logger.Debugw("Debug variadic should be filtered", "key", "value")
	logger.Infow("Info variadic should be filtered", "key", "value")

	// These should pass (at or above warn level)
	logger.Warn("Warn should pass")
	logger.Error("Error should pass")
	logger.Warnf("Warn formatted should pass")
	logger.Errorf("Error formatted should pass")
	logger.Warnw("Warn variadic should pass", "key", "value")
	logger.Errorw("Error variadic should pass", "key", "value")

	// Test using Log methods
	logger.Log(DebugLevel, "Log debug should be filtered")
	logger.Log(InfoLevel, "Log info should be filtered")
	logger.Log(WarnLevel, "Log warn should pass")
	logger.Log(ErrorLevel, "Log error should pass")

	logger.Logf(DebugLevel, "Logf debug should be filtered")
	logger.Logf(InfoLevel, "Logf info should be filtered")
	logger.Logf(WarnLevel, "Logf warn should pass")
	logger.Logf(ErrorLevel, "Logf error should pass")

	// Read log file and verify only warn/error messages appear
	if content, err := os.ReadFile(logFile); err != nil {
		t.Errorf("Failed to read log file: %v", err)
	} else {
		contentStr := string(content)
		// Should not contain debug/info messages
		if strings.Contains(contentStr, "should be filtered") {
			t.Error("Filtered messages found in log file")
		}
		// Should contain warn/error messages
		if !strings.Contains(contentStr, "should pass") {
			t.Error("Expected messages not found in log file")
		}
	}

	os.Remove(logFile)
}

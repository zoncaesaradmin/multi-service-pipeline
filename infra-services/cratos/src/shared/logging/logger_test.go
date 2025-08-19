package logging

import (
	"os"
	"strings"
	"testing"
)

const (
	testLogFile          = "/tmp/test.log"
	testLoggerName       = "test-logger"
	testServiceName      = "test-service"
	testComponentName    = "test"
	newLoggerErrorFmt    = "NewLoggerWithConfig() error = %v"
	logFileNotCreatedFmt = "Log file was not created: %v"
	logFileEmptyMsg      = "Log file is empty"

	// Error message constants
	errFilenameRequired   = "filename is required"
	errLoggerNameRequired = "logger name is required"
	errComponentRequired  = "component name is required"
	errServiceRequired    = "service name is required"

	// Test assertion constants
	getLevelErrorFmt = "GetLevel() = %v, want %v"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
		{PanicLevel, "PANIC"},
		{Level(999), "UNKNOWN"},
		{Level(-1), "UNKNOWN"},  // Negative level
		{Level(100), "UNKNOWN"}, // High level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoggerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: LoggerConfig{
				Level:         InfoLevel,
				FilePath:      testLogFile,
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			wantErr: false,
		},
		{
			name: "missing filename",
			config: LoggerConfig{
				Level:         InfoLevel,
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			wantErr: true,
			errMsg:  errFilenameRequired,
		},
		{
			name: "empty filename",
			config: LoggerConfig{
				Level:         InfoLevel,
				FilePath:      "",
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			wantErr: true,
			errMsg:  errFilenameRequired,
		},
		{
			name: "missing logger name",
			config: LoggerConfig{
				Level:         InfoLevel,
				FilePath:      testLogFile,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			wantErr: true,
			errMsg:  errLoggerNameRequired,
		},
		{
			name: "empty logger name",
			config: LoggerConfig{
				Level:         InfoLevel,
				FilePath:      testLogFile,
				LoggerName:    "",
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			wantErr: true,
			errMsg:  errLoggerNameRequired,
		},
		{
			name: "missing service name",
			config: LoggerConfig{
				Level:         InfoLevel,
				FilePath:      testLogFile,
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
			},
			wantErr: true,
			errMsg:  errServiceRequired,
		},
		{
			name: "empty service name",
			config: LoggerConfig{
				Level:         InfoLevel,
				FilePath:      testLogFile,
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   "",
			},
			wantErr: true,
			errMsg:  errServiceRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoggerConfig.Validate() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("LoggerConfig.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("LoggerConfig.Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test all fields of default config
	if config.Level != InfoLevel {
		t.Errorf("DefaultConfig() Level = %v, want %v", config.Level, InfoLevel)
	}

	if config.LoggerName != "default" {
		t.Errorf("DefaultConfig() LoggerName = %v, want %v", config.LoggerName, "default")
	}

	if config.ComponentName != "application" {
		t.Errorf("DefaultConfig() ComponentName = %v, want %v", config.ComponentName, "application")
	}

	if config.ServiceName != "service" {
		t.Errorf("DefaultConfig() ServiceName = %v, want %v", config.ServiceName, "service")
	}

	if config.FilePath != "/tmp/app.log" {
		t.Errorf("DefaultConfig() FilePath = %v, want %v", config.FilePath, "/tmp/app.log")
	}

	// Test that default config is valid
	if err := config.Validate(); err != nil {
		t.Errorf("DefaultConfig() produced invalid config: %v", err)
	}
}

func TestKeysAndValuesToFields(t *testing.T) {
	tests := []struct {
		name           string
		keysAndValues  []interface{}
		expectedFields Fields
	}{
		{
			name:           "empty",
			keysAndValues:  []interface{}{},
			expectedFields: Fields{},
		},
		{
			name:           "single pair",
			keysAndValues:  []interface{}{"key", "value"},
			expectedFields: Fields{"key": "value"},
		},
		{
			name:           "multiple pairs",
			keysAndValues:  []interface{}{"key1", "value1", "key2", 42, "key3", true},
			expectedFields: Fields{"key1": "value1", "key2": 42, "key3": true},
		},
		{
			name:           "odd number of arguments",
			keysAndValues:  []interface{}{"key1", "value1", "key2"},
			expectedFields: Fields{"key1": "value1"},
		},
		{
			name:           "single key no value",
			keysAndValues:  []interface{}{"lonely_key"},
			expectedFields: Fields{},
		},
		{
			name:           "mixed types as keys",
			keysAndValues:  []interface{}{123, "number_key", true, "bool_key", 3.14, "float_key"},
			expectedFields: Fields{"123": "number_key", "true": "bool_key", "3.14": "float_key"},
		},
		{
			name:           "nil values",
			keysAndValues:  []interface{}{"key1", nil, "key2", "value2"},
			expectedFields: Fields{"key1": nil, "key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := keysAndValuesToFields(tt.keysAndValues...)

			if len(result) != len(tt.expectedFields) {
				t.Errorf("Expected %d fields, got %d", len(tt.expectedFields), len(result))
			}

			for key, expectedValue := range tt.expectedFields {
				if result[key] != expectedValue {
					t.Errorf("Expected field %s=%v, got %v", key, expectedValue, result[key])
				}
			}
		})
	}
}

func TestNewLoggerWithValidConfig(t *testing.T) {
	logFile := "/tmp/test_new_logger_valid.log"
	defer func() {
		_ = os.Remove(logFile)
	}()

	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      logFile,
		LoggerName:    "valid-test-logger",
		ComponentName: "valid-test",
		ServiceName:   "valid-test-service",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Test basic functionality
	if logger.GetLevel() != InfoLevel {
		t.Errorf(getLevelErrorFmt, logger.GetLevel(), InfoLevel)
	}

	// Test logging
	logger.Info("Test message from valid config")
	logger.Warn("Warning message from valid config")
	logger.Error("Error message from valid config")

	// Check if log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf(logFileNotCreatedFmt, err)
	}
}

func TestNewLoggerWithNilConfig(t *testing.T) {
	// Test with nil config - should use default config
	logger, err := NewLogger(nil)
	if err != nil {
		t.Fatalf("NewLogger(nil) error = %v", err)
	}
	defer logger.Close()

	// Should use default config values
	if logger.GetLevel() != InfoLevel {
		t.Errorf(getLevelErrorFmt, logger.GetLevel(), InfoLevel)
	}

	// Test that it actually works
	logger.Info("Test message with nil config")
}

func TestNewLoggerWithInvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *LoggerConfig
		expectError string
	}{
		{
			name: "missing filename",
			config: &LoggerConfig{
				Level:         InfoLevel,
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			expectError: errFilenameRequired,
		},
		{
			name: "empty filename",
			config: &LoggerConfig{
				Level:         InfoLevel,
				FilePath:      "",
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			expectError: errFilenameRequired,
		},
		{
			name: "missing logger name",
			config: &LoggerConfig{
				Level:         InfoLevel,
				FilePath:      testLogFile,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			expectError: errLoggerNameRequired,
		},
		{
			name: "missing service name",
			config: &LoggerConfig{
				Level:         InfoLevel,
				FilePath:      testLogFile,
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
			},
			expectError: errServiceRequired,
		},
		{
			name: "invalid file path",
			config: &LoggerConfig{
				Level:         InfoLevel,
				FilePath:      "/invalid/nonexistent/path/test.log",
				LoggerName:    testLoggerName,
				ComponentName: testComponentName,
				ServiceName:   testServiceName,
			},
			expectError: "failed to create logger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if err == nil {
				if logger != nil {
					logger.Close()
				}
				t.Errorf("NewLogger() expected error for %s but got none", tt.name)
				return
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("NewLogger() error = %v, want error containing %v", err, tt.expectError)
			}
		})
	}
}

func TestNewLoggerConfigValidationPath(t *testing.T) {
	// Test that validation error is properly wrapped
	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      "", // Invalid - will trigger validation error
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLogger(config)
	if err == nil {
		if logger != nil {
			logger.Close()
		}
		t.Error("NewLogger() expected validation error but got none")
		return
	}

	// Check that the error is properly wrapped
	if !strings.Contains(err.Error(), "invalid logger configuration") {
		t.Errorf("NewLogger() error = %v, want error containing 'invalid logger configuration'", err)
	}

	if !strings.Contains(err.Error(), errFilenameRequired) {
		t.Errorf("NewLogger() error = %v, want error containing %v", err, errFilenameRequired)
	}
}

func TestNewLoggerZerologCreationFailure(t *testing.T) {
	// Test the path where NewLoggerWithConfig fails
	config := &LoggerConfig{
		Level:         InfoLevel,
		FilePath:      "/root/impossible/path/test.log", // Path that should fail
		LoggerName:    testLoggerName,
		ComponentName: testComponentName,
		ServiceName:   testServiceName,
	}

	logger, err := NewLogger(config)
	if err == nil {
		if logger != nil {
			logger.Close()
		}
		t.Error("NewLogger() expected error for impossible path but got none")
		return
	}

	// Check that the error is properly wrapped
	if !strings.Contains(err.Error(), "failed to create logger") {
		t.Errorf("NewLogger() error = %v, want error containing 'failed to create logger'", err)
	}
}

func TestNewLoggerWithDifferentLevels(t *testing.T) {
	// Test NewLogger with all different log levels
	levels := []Level{DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel, PanicLevel}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			logFile := "/tmp/test_new_logger_" + level.String() + ".log"
			defer func() {
				_ = os.Remove(logFile)
			}()

			config := &LoggerConfig{
				Level:         level,
				FilePath:      logFile,
				LoggerName:    "level-test-logger",
				ComponentName: "level-test",
				ServiceName:   "level-test-service",
			}

			logger, err := NewLogger(config)
			if err != nil {
				t.Fatalf("NewLogger() with level %s error = %v", level.String(), err)
			}
			defer logger.Close()

			// Test that the level is set correctly
			if logger.GetLevel() != level {
				t.Errorf(getLevelErrorFmt, logger.GetLevel(), level)
			}

			// Test basic logging
			logger.Info("Test message with level " + level.String())
		})
	}
}

func TestFieldsType(t *testing.T) {
	// Test that Fields type works as expected
	fields := Fields{
		"string":  "value",
		"int":     42,
		"bool":    true,
		"float":   3.14,
		"nil":     nil,
		"complex": map[string]interface{}{"nested": "value"},
	}

	// Test accessing fields
	if fields["string"] != "value" {
		t.Errorf("Expected string field to be 'value', got %v", fields["string"])
	}

	if fields["int"] != 42 {
		t.Errorf("Expected int field to be 42, got %v", fields["int"])
	}

	if fields["bool"] != true {
		t.Errorf("Expected bool field to be true, got %v", fields["bool"])
	}

	// Test that Fields can be used as map[string]interface{}
	var m map[string]interface{} = fields
	if m["string"] != "value" {
		t.Errorf("Expected Fields to work as map[string]interface{}")
	}
}

func TestKeysAndValuesToFieldsEdgeCases(t *testing.T) {
	// Test edge cases for keysAndValuesToFields function

	// Test with zero arguments
	result := keysAndValuesToFields()
	if len(result) != 0 {
		t.Errorf("Expected empty fields for no arguments, got %d fields", len(result))
	}

	// Test with nil arguments
	result = keysAndValuesToFields(nil, nil)
	if len(result) != 1 || result["<nil>"] != nil {
		t.Errorf("Expected one field with nil key and value")
	}

	// Test with large number of arguments
	result = keysAndValuesToFields("k1", "v1", "k2", "v2", "k3", "v3", "k4", "v4", "k5", "v5")
	if len(result) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(result))
	}
}

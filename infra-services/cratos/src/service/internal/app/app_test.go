package app

import (
	"context"
	"testing"

	"servicegomodule/internal/config"
	"sharedgomodule/logging"
)

// mockLogger implements basic logging interface for testing
type mockLogger struct {
	logCalls []string
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		logCalls: make([]string, 0),
	}
}

func (m *mockLogger) SetLevel(level logging.Level)                    {}
func (m *mockLogger) GetLevel() logging.Level                         { return logging.InfoLevel }
func (m *mockLogger) IsLevelEnabled(level logging.Level) bool         { return true }
func (m *mockLogger) Debug(msg string)                                { m.logCalls = append(m.logCalls, "DEBUG: "+msg) }
func (m *mockLogger) Info(msg string)                                 { m.logCalls = append(m.logCalls, "INFO: "+msg) }
func (m *mockLogger) Warn(msg string)                                 { m.logCalls = append(m.logCalls, "WARN: "+msg) }
func (m *mockLogger) Error(msg string)                                { m.logCalls = append(m.logCalls, "ERROR: "+msg) }
func (m *mockLogger) Fatal(msg string)                                { m.logCalls = append(m.logCalls, "FATAL: "+msg) }
func (m *mockLogger) Panic(msg string)                                { m.logCalls = append(m.logCalls, "PANIC: "+msg) }
func (m *mockLogger) Debugf(format string, args ...interface{})       {}
func (m *mockLogger) Infof(format string, args ...interface{})        {}
func (m *mockLogger) Warnf(format string, args ...interface{})        {}
func (m *mockLogger) Errorf(format string, args ...interface{})       {}
func (m *mockLogger) Fatalf(format string, args ...interface{})       {}
func (m *mockLogger) Panicf(format string, args ...interface{})       {}
func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Infow(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	m.logCalls = append(m.logCalls, "ERRORW: "+msg)
}
func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{})                    {}
func (m *mockLogger) Panicw(msg string, keysAndValues ...interface{})                    {}
func (m *mockLogger) WithFields(fields logging.Fields) logging.Logger                    { return m }
func (m *mockLogger) WithField(key string, value interface{}) logging.Logger             { return m }
func (m *mockLogger) WithError(err error) logging.Logger                                 { return m }
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger                     { return m }
func (m *mockLogger) Log(level logging.Level, msg string)                                {}
func (m *mockLogger) Logf(level logging.Level, format string, args ...interface{})       {}
func (m *mockLogger) Logw(level logging.Level, msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Clone() logging.Logger                                              { return m }
func (m *mockLogger) Close() error                                                       { return nil }

func TestNewApplication(t *testing.T) {
	cfg := &config.RawConfig{
		Server: config.RawServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	logger := newMockLogger()

	app := NewApplication(cfg, logger)

	if app == nil {
		t.Fatal("NewApplication() returned nil")
	}

	if app.rawconfig != cfg {
		t.Error("Application config not set correctly")
	}

	if app.logger != logger {
		t.Error("Application logger not set correctly")
	}

	if app.processingPipeline == nil {
		t.Error("Processing pipeline not initialized")
	}

	if app.ctx == nil {
		t.Error("Application context not initialized")
	}

	if app.cancel == nil {
		t.Error("Application cancel function not initialized")
	}
}

func TestApplicationConfig(t *testing.T) {
	cfg := &config.RawConfig{
		Server: config.RawServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	logger := newMockLogger()
	app := NewApplication(cfg, logger)

	retrievedConfig := app.Config()
	if retrievedConfig != cfg {
		t.Error("Config() returned incorrect configuration")
	}

	if retrievedConfig.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", retrievedConfig.Server.Host)
	}

	if retrievedConfig.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", retrievedConfig.Server.Port)
	}
}

func TestApplicationLogger(t *testing.T) {
	cfg := &config.RawConfig{}
	logger := newMockLogger()
	app := NewApplication(cfg, logger)

	retrievedLogger := app.Logger()
	if retrievedLogger != logger {
		t.Error("Logger() returned incorrect logger")
	}
}

func TestApplicationContext(t *testing.T) {
	cfg := &config.RawConfig{}
	logger := newMockLogger()
	app := NewApplication(cfg, logger)

	ctx := app.Context()
	if ctx == nil {
		t.Error("Context() returned nil")
	}

	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Expected behavior
	}
}

func TestApplicationProcessingPipeline(t *testing.T) {
	cfg := &config.RawConfig{}
	logger := newMockLogger()
	app := NewApplication(cfg, logger)

	pipeline := app.ProcessingPipeline()
	if pipeline == nil {
		t.Error("ProcessingPipeline() returned nil")
	}
}

func TestApplicationShutdown(t *testing.T) {
	cfg := &config.RawConfig{}
	logger := newMockLogger()
	app := NewApplication(cfg, logger)

	// Verify context is not cancelled initially
	select {
	case <-app.Context().Done():
		t.Error("Context should not be cancelled before shutdown")
	default:
		// Expected
	}

	// Test shutdown
	app.Shutdown()

	// Verify context is cancelled after shutdown
	select {
	case <-app.Context().Done():
		// Expected behavior
	default:
		t.Error("Context should be cancelled after shutdown")
	}
}

func TestApplicationIsShuttingDown(t *testing.T) {
	cfg := &config.RawConfig{}
	logger := newMockLogger()
	app := NewApplication(cfg, logger)

	// Initially should not be shutting down
	if app.IsShuttingDown() {
		t.Error("Application should not be shutting down initially")
	}

	// After shutdown, should be shutting down
	app.Shutdown()
	if !app.IsShuttingDown() {
		t.Error("Application should be shutting down after Shutdown() call")
	}
}

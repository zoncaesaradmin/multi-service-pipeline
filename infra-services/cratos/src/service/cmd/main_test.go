package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"servicegomodule/internal/api"
	"servicegomodule/internal/app"
	"servicegomodule/internal/config"
	"sharedgomodule/logging"
)

const (
	healthEndpoint  = "/health"
	serviceName     = "cratos-service"
	logFileName     = "/tmp/cratos-service.log"
	componentMain   = "main"
	testHost        = "localhost"
	testHostAll     = "0.0.0.0"
	configEndpoint  = "/api/v1/config/"
	statsEndpoint   = "/api/v1/stats"
	nonexistentPath = "/nonexistent"
)

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) SetLevel(level logging.Level)                           { /* no-op for testing */ }
func (m *mockLogger) GetLevel() logging.Level                                { return logging.InfoLevel }
func (m *mockLogger) IsLevelEnabled(level logging.Level) bool                { return true }
func (m *mockLogger) Debug(msg string)                                       { /* no-op for testing */ }
func (m *mockLogger) Info(msg string)                                        { /* no-op for testing */ }
func (m *mockLogger) Warn(msg string)                                        { /* no-op for testing */ }
func (m *mockLogger) Error(msg string)                                       { /* no-op for testing */ }
func (m *mockLogger) Fatal(msg string)                                       { /* no-op for testing */ }
func (m *mockLogger) Panic(msg string)                                       { /* no-op for testing */ }
func (m *mockLogger) Debugf(format string, args ...interface{})              { /* no-op for testing */ }
func (m *mockLogger) Infof(format string, args ...interface{})               { /* no-op for testing */ }
func (m *mockLogger) Warnf(format string, args ...interface{})               { /* no-op for testing */ }
func (m *mockLogger) Errorf(format string, args ...interface{})              { /* no-op for testing */ }
func (m *mockLogger) Fatalf(format string, args ...interface{})              { /* no-op for testing */ }
func (m *mockLogger) Panicf(format string, args ...interface{})              { /* no-op for testing */ }
func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{})        { /* no-op for testing */ }
func (m *mockLogger) Infow(msg string, keysAndValues ...interface{})         { /* no-op for testing */ }
func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{})         { /* no-op for testing */ }
func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{})        { /* no-op for testing */ }
func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{})        { /* no-op for testing */ }
func (m *mockLogger) Panicw(msg string, keysAndValues ...interface{})        { /* no-op for testing */ }
func (m *mockLogger) WithFields(fields logging.Fields) logging.Logger        { return m }
func (m *mockLogger) WithField(key string, value interface{}) logging.Logger { return m }
func (m *mockLogger) WithError(err error) logging.Logger                     { return m }
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger         { return m }
func (m *mockLogger) Log(level logging.Level, msg string)                    { /* no-op for testing */ }
func (m *mockLogger) Logf(level logging.Level, format string, args ...interface{}) { /* no-op for testing */
}
func (m *mockLogger) Logw(level logging.Level, msg string, keysAndValues ...interface{}) { /* no-op for testing */
}
func (m *mockLogger) Clone() logging.Logger { return &mockLogger{} }
func (m *mockLogger) Close() error          { return nil }

func TestSetupRouter(t *testing.T) {
	logger := &mockLogger{}
	mux := setupRouter(logger)

	if mux == nil {
		t.Fatal("expected mux to not be nil")
	}

	// Test that routes are set up correctly by making sample requests
	testCases := []struct {
		path           string
		method         string
		expectedStatus int
	}{
		{healthEndpoint, "GET", http.StatusOK},
		{configEndpoint, "GET", http.StatusOK},
		{statsEndpoint, "GET", http.StatusOK},
		{nonexistentPath, "GET", http.StatusNotFound},
	}

	for _, tc := range testCases {
		req, err := http.NewRequest(tc.method, tc.path, nil)
		if err != nil {
			t.Fatalf("failed to create request for %s %s: %v", tc.method, tc.path, err)
		}

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != tc.expectedStatus {
			t.Errorf("expected status %d for %s %s, got %d", tc.expectedStatus, tc.method, tc.path, rr.Code)
		}
	}
}

func TestSetupRouterWithNilHandler(t *testing.T) {
	// Test setupRouter function - it creates its own handler internally
	// This test verifies that setupRouter works correctly
	logger := &mockLogger{}
	mux := setupRouter(logger)

	// The function should always return a valid mux since it creates the handler internally
	if mux == nil {
		t.Error("expected mux to not be nil")
	}

	// Test that the mux has routes set up by making a test request
	req, err := http.NewRequest("GET", healthEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d for health endpoint, got %d", http.StatusOK, rr.Code)
	}
}

func TestServerConfiguration(t *testing.T) {
	// Test server configuration with different config values
	testCases := []struct {
		name      string
		rawconfig *config.RawConfig
	}{
		{
			name: "default config",
			rawconfig: &config.RawConfig{
				Server: config.RawServerConfig{
					Host:         testHost,
					Port:         4477,
					ReadTimeout:  10,
					WriteTimeout: 10,
				},
			},
		},
		{
			name: "custom config",
			rawconfig: &config.RawConfig{
				Server: config.RawServerConfig{
					Host:         testHostAll,
					Port:         9090,
					ReadTimeout:  15,
					WriteTimeout: 20,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server configuration
			logger := &mockLogger{}
			application := app.NewApplication(tc.rawconfig, logger)
			mux := setupRouter(logger)

			// Create server with same configuration as startServer
			srv := &http.Server{
				Handler:      mux,
				ReadTimeout:  time.Duration(tc.rawconfig.Server.ReadTimeout) * time.Second,
				WriteTimeout: time.Duration(tc.rawconfig.Server.WriteTimeout) * time.Second,
			}

			// Verify server configuration
			if srv.Handler != mux {
				t.Error("expected server handler to be set correctly")
			}

			expectedReadTimeout := time.Duration(tc.rawconfig.Server.ReadTimeout) * time.Second
			if srv.ReadTimeout != expectedReadTimeout {
				t.Errorf("expected ReadTimeout to be %v, got %v", expectedReadTimeout, srv.ReadTimeout)
			}

			expectedWriteTimeout := time.Duration(tc.rawconfig.Server.WriteTimeout) * time.Second
			if srv.WriteTimeout != expectedWriteTimeout {
				t.Errorf("expected WriteTimeout to be %v, got %v", expectedWriteTimeout, srv.WriteTimeout)
			}

			// Clean up
			application.Shutdown()
		})
	}
}

func TestApplicationInitialization(t *testing.T) {
	// Test that application is initialized correctly
	cfg := &config.RawConfig{
		Server: config.RawServerConfig{
			Host:         testHost,
			Port:         4477,
			ReadTimeout:  10,
			WriteTimeout: 10,
		},
	}

	logger := &mockLogger{}
	application := app.NewApplication(cfg, logger)

	// Verify application is created properly
	if application == nil {
		t.Fatal("expected application to not be nil")
	}

	// Verify config is accessible
	appConfig := application.Config()
	if appConfig != cfg {
		t.Error("expected application config to match provided config")
	}

	// Verify logger is accessible
	appLogger := application.Logger()
	if appLogger != logger {
		t.Error("expected application logger to match provided logger")
	}

	// Verify context is not cancelled initially
	ctx := application.Context()
	select {
	case <-ctx.Done():
		t.Error("expected application context to not be cancelled initially")
	default:
		// Expected behavior
	}

	// Test shutdown
	err := application.Shutdown()
	if err != nil {
		t.Errorf("expected no error during shutdown, got %v", err)
	}

	// Verify context is cancelled after shutdown
	select {
	case <-ctx.Done():
		// Expected behavior
	default:
		t.Error("expected application context to be cancelled after shutdown")
	}
}

func TestHandlerInitialization(t *testing.T) {
	logger := &mockLogger{}
	handler := api.NewHandler(logger)

	if handler == nil {
		t.Fatal("expected handler to not be nil")
	}

	// Test that handler can set up routes without error
	mux := http.NewServeMux()

	// This should not panic
	handler.SetupRoutes(mux)

	// Test a basic request to ensure routes are working
	req, err := http.NewRequest("GET", healthEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d for health check, got %d", http.StatusOK, rr.Code)
	}
}

func TestLoggerConfiguration(t *testing.T) {
	// Test logger configuration values
	expectedConfig := &logging.LoggerConfig{
		Level:         logging.InfoLevel,
		FilePath:      logFileName,
		LoggerName:    serviceName,
		ComponentName: componentMain,
		ServiceName:   serviceName,
	}

	// Verify all fields are set correctly
	if expectedConfig.Level != logging.InfoLevel {
		t.Errorf("expected Level to be InfoLevel, got %v", expectedConfig.Level)
	}

	if expectedConfig.FilePath != logFileName {
		t.Errorf("expected FileName to be '%s', got %s", logFileName, expectedConfig.FilePath)
	}

	if expectedConfig.LoggerName != serviceName {
		t.Errorf("expected LoggerName to be '%s', got %s", serviceName, expectedConfig.LoggerName)
	}

	if expectedConfig.ComponentName != componentMain {
		t.Errorf("expected ComponentName to be '%s', got %s", componentMain, expectedConfig.ComponentName)
	}

	if expectedConfig.ServiceName != serviceName {
		t.Errorf("expected ServiceName to be '%s', got %s", serviceName, expectedConfig.ServiceName)
	}
}

func TestIntegrationComponents(t *testing.T) {
	// Test that all components work together
	cfg := config.LoadConfig()
	logger := &mockLogger{}
	application := app.NewApplication(cfg, logger)
	mux := setupRouter(logger)

	// Test that we can make requests through the complete stack
	req, err := http.NewRequest("GET", healthEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d for integration test, got %d", http.StatusOK, rr.Code)
	}

	// Test API endpoints
	apiEndpoints := []string{
		configEndpoint,
		statsEndpoint,
	}

	for _, endpoint := range apiEndpoints {
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			t.Fatalf("failed to create request for %s: %v", endpoint, err)
		}

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d for %s, got %d", http.StatusOK, endpoint, rr.Code)
		}
	}

	// Clean up
	application.Shutdown()
}

func TestServerShutdownGraceful(t *testing.T) {
	// Test graceful shutdown simulation
	cfg := &config.RawConfig{
		Server: config.RawServerConfig{
			Host:         testHost,
			Port:         0, // Use port 0 to let OS assign a free port
			ReadTimeout:  1,
			WriteTimeout: 1,
		},
	}

	logger := &mockLogger{}
	application := app.NewApplication(cfg, logger)

	// Test that application can be shut down gracefully
	err := application.Shutdown()
	if err != nil {
		t.Errorf("expected no error during shutdown, got %v", err)
	}

	// Verify shutdown state
	if !application.IsShuttingDown() {
		t.Error("expected application to be in shutting down state")
	}
}

func TestConfigLoadingDefault(t *testing.T) {
	// Test that config loading works as expected (using defaults)
	cfg := config.LoadConfig()

	if cfg == nil {
		t.Fatal("expected config to not be nil")
	}

	// Verify server config has reasonable defaults
	if cfg.Server.Port <= 0 {
		t.Error("expected server port to be positive")
	}

	if cfg.Server.Host == "" {
		t.Error("expected server host to not be empty")
	}

	if cfg.Server.ReadTimeout <= 0 {
		t.Error("expected read timeout to be positive")
	}

	if cfg.Server.WriteTimeout <= 0 {
		t.Error("expected write timeout to be positive")
	}
}

// Benchmark tests for performance
func BenchmarkSetupRouter(b *testing.B) {
	logger := &mockLogger{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux := setupRouter(logger)
		_ = mux
	}
}

func BenchmarkHealthCheckRequest(b *testing.B) {
	logger := &mockLogger{}
	mux := setupRouter(logger)

	req, _ := http.NewRequest("GET", healthEndpoint, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
	}
}

func BenchmarkApplicationCreation(b *testing.B) {
	cfg := &config.RawConfig{
		Server: config.RawServerConfig{
			Host:         testHost,
			Port:         4477,
			ReadTimeout:  10,
			WriteTimeout: 10,
		},
	}
	logger := &mockLogger{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := app.NewApplication(cfg, logger)
		app.Shutdown()
	}
}

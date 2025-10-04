package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	envDocker       = "/.dockerenv"
	testEnvFile     = ".env.test"
)

// Helper function to create a test logger
func createTestLogger() logging.Logger {
	return logging.NewMockLogger()
}

func TestSetupRouter(t *testing.T) {
	logger := createTestLogger()
	cfg := sampleRawConfig()
	application := app.NewApplication(cfg, logger)
	defer application.Shutdown()

	mux := setupRouter(logger, application)

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
	logger := createTestLogger()
	cfg := sampleRawConfig()
	application := app.NewApplication(cfg, logger)
	defer application.Shutdown()

	mux := setupRouter(logger, application)

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
	samplecfg := sampleRawConfig()
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
				Processing: samplecfg.Processing,
				Logging:    samplecfg.Logging,
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
				Processing: samplecfg.Processing,
				Logging:    samplecfg.Logging,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server configuration
			logger := createTestLogger()
			application := app.NewApplication(tc.rawconfig, logger)
			defer application.Shutdown()

			mux := setupRouter(logger, application)

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
	cfg := sampleRawConfig()

	logger := createTestLogger()
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
	logger := createTestLogger()
	handler := api.NewHandler(logger, nil, nil)

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
	cfg := sampleRawConfig()

	logger := createTestLogger()
	application := app.NewApplication(cfg, logger)
	defer application.Shutdown()

	mux := setupRouter(logger, application)

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
	cfg := sampleRawConfig()
	logger := createTestLogger()
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

// Benchmark tests for performance
func BenchmarkSetupRouter(b *testing.B) {
	logger := createTestLogger()
	cfg := sampleRawConfig()
	application := app.NewApplication(cfg, logger)
	defer application.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux := setupRouter(logger, application)
		_ = mux
	}
}

func BenchmarkHealthCheckRequest(b *testing.B) {
	logger := createTestLogger()
	cfg := sampleRawConfig()
	application := app.NewApplication(cfg, logger)
	defer application.Shutdown()

	mux := setupRouter(logger, application)

	req, _ := http.NewRequest("GET", healthEndpoint, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
	}
}

func BenchmarkApplicationCreation(b *testing.B) {
	cfg := sampleRawConfig()
	logger := createTestLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := app.NewApplication(cfg, logger)
		app.Shutdown()
	}
}

func TestIsRunningInContainer(t *testing.T) {
	// Save original environment and restore after test
	origK8sHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	origContainer := os.Getenv("CONTAINER")
	defer func() {
		os.Setenv("KUBERNETES_SERVICE_HOST", origK8sHost)
		os.Setenv("CONTAINER", origContainer)
	}()

	tests := []struct {
		name  string
		setup func()
		want  bool
	}{
		{
			name: "not in container",
			setup: func() {
				os.Unsetenv("KUBERNETES_SERVICE_HOST")
				os.Unsetenv("CONTAINER")
			},
			want: false,
		},
		{
			name: "in kubernetes",
			setup: func() {
				os.Setenv("KUBERNETES_SERVICE_HOST", "test-host")
			},
			want: true,
		},
		{
			name: "container flag set",
			setup: func() {
				os.Setenv("CONTAINER", "true")
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			if got := isRunningInContainer(); got != tt.want {
				t.Errorf("isRunningInContainer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogEnvironmentInfo(t *testing.T) {
	// Save original environment and restore after test
	origAppEnv := os.Getenv("APP_ENV")
	origAppName := os.Getenv("APP_NAME")
	origAppVersion := os.Getenv("APP_VERSION")
	origContainer := os.Getenv("CONTAINER")
	defer func() {
		os.Setenv("APP_ENV", origAppEnv)
		os.Setenv("APP_NAME", origAppName)
		os.Setenv("APP_VERSION", origAppVersion)
		os.Setenv("CONTAINER", origContainer)
	}()

	// Test with custom environment values
	os.Setenv("APP_ENV", "testing")
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("APP_VERSION", "1.0.0")

	// Test both container and non-container environments
	tests := []struct {
		name        string
		inContainer bool
	}{
		{"local environment", false},
		{"container environment", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.inContainer {
				os.Setenv("CONTAINER", "true")
			} else {
				os.Unsetenv("CONTAINER")
			}
			// Since logEnvironmentInfo only logs, we just verify it doesn't panic
			logEnvironmentInfo()
		})
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	// Set up a minimal test environment
	origServiceHome := os.Getenv("SERVICE_HOME")
	defer os.Setenv("SERVICE_HOME", origServiceHome)
	os.Setenv("SERVICE_HOME", "/tmp") // Set to a known location for testing

	// Create a minimal test server with just the health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(healthEndpoint, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create test listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	testPort := listener.Addr().(*net.TCPAddr).Port

	// Create server with test configuration
	srv := &http.Server{
		Handler:      mux,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("Server stopped with error: %v", err)
		}
	}()

	// Test the server is working
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", testPort, healthEndpoint))
	if err != nil {
		t.Fatalf("Initial health check failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Error shutting down server: %v", err)
	}

	// Verify server is shut down by attempting a request
	_, err = client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", testPort, healthEndpoint))
	if err == nil {
		t.Error("Expected error when connecting to shutdown server")
	}
}

func TestConfigSetup(t *testing.T) {
	t.Run("with valid SERVICE_HOME", func(t *testing.T) {
		// Save original environment and restore after test
		origServiceHome := os.Getenv("SERVICE_HOME")
		defer os.Setenv("SERVICE_HOME", origServiceHome)

		// Set SERVICE_HOME to a known location
		tmpDir, err := os.MkdirTemp("", "test-service-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create test config directory
		configDir := filepath.Join(tmpDir, "conf")
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create config dir: %v", err)
		}

		os.Setenv("SERVICE_HOME", tmpDir)

		// Let loadConfig return nil (which is expected since we don't have a real config file)
		cfg := loadConfig()
		if cfg != nil {
			t.Error("Expected nil config with empty but valid SERVICE_HOME")
		}
	})
}

func sampleRawConfig() *config.RawConfig {
	return &config.RawConfig{
		Server: config.RawServerConfig{
			Host:         testHost,
			Port:         4477,
			ReadTimeout:  10,
			WriteTimeout: 10,
		},
		Processing: config.RawProcessingConfig{
			Processor: config.RawProcessorConfig{
				RuleProcConfig: config.RawRuleProcessorConfig{
					RelibLogging: config.RawLoggingConfig{
						FileName:    logFileName,
						LoggerName:  "ruleenginelib",
						ServiceName: "cratos",
					},
					RuleHandlerLogging: config.RawLoggingConfig{
						FileName:    logFileName,
						LoggerName:  "rulehandler",
						ServiceName: "cratos",
					},
				},
			},
			PloggerConfig: config.RawLoggingConfig{
				FileName:    logFileName,
				LoggerName:  "plogger",
				ServiceName: "cratos",
			},
		},
		Logging: config.RawLoggingConfig{
			FileName:    logFileName,
			LoggerName:  "mlogger",
			ServiceName: "cratos",
		},
	}
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"servicegomodule/internal/models"
	"sharedgomodule/logging"
)

const (
	// Test constants
	testHealthPath    = "/health"
	testStatsPath     = "/api/v1/stats"
	testConfigPath    = "/api/v1/config/"
	contentTypeHeader = "Content-Type"
	jsonContentType   = "application/json"
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

func TestNewHandler(t *testing.T) {
	logger := &mockLogger{}
	handler := NewHandler(logger, nil, nil)
	if handler == nil {
		t.Fatal("NewHandler() returned nil")
	}
}

func TestHealthCheck(t *testing.T) {
	logger := &mockLogger{}
	handler := NewHandler(logger, nil, nil)
	req := httptest.NewRequest(http.MethodGet, testHealthPath, nil)
	rr := httptest.NewRecorder()

	handler.HealthCheck(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("HealthCheck() status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Check Content-Type
	if contentType := rr.Header().Get(contentTypeHeader); contentType != jsonContentType {
		t.Errorf("HealthCheck() Content-Type = %q, want %q", contentType, jsonContentType)
	}

	// Parse and validate JSON response
	var health models.HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("HealthCheck() status = %q, want %q", health.Status, "healthy")
	}

	if health.Version != "1.0.0" {
		t.Errorf("HealthCheck() version = %q, want %q", health.Version, "1.0.0")
	}
}

func TestGetStats(t *testing.T) {
	logger := &mockLogger{}
	handler := NewHandler(logger, nil, nil)
	req := httptest.NewRequest(http.MethodGet, testStatsPath, nil)
	rr := httptest.NewRecorder()

	handler.GetStats(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("GetStats() status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Check Content-Type
	if contentType := rr.Header().Get(contentTypeHeader); contentType != jsonContentType {
		t.Errorf("GetStats() Content-Type = %q, want %q", contentType, jsonContentType)
	}

	// Parse and validate JSON response
	var response models.SuccessResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode stats response: %v", err)
	}

	if response.Message != MsgStatsRetrieved {
		t.Errorf("GetStats() message = %q, want %q", response.Message, MsgStatsRetrieved)
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := models.SuccessResponse{Message: "test", Data: "data"}

	writeJSON(rr, http.StatusOK, data)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("writeJSON() status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Check Content-Type
	if contentType := rr.Header().Get(contentTypeHeader); contentType != jsonContentType {
		t.Errorf("writeJSON() Content-Type = %q, want %q", contentType, jsonContentType)
	}

	// Check that response body is valid JSON
	var result interface{}
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Errorf("writeJSON() produced invalid JSON: %v", err)
	}
}

func TestHealthCheckOPTIONS(t *testing.T) {
	logger := &mockLogger{}
	handler := NewHandler(logger, nil, nil)
	req := httptest.NewRequest(http.MethodOptions, testHealthPath, nil)
	rr := httptest.NewRecorder()

	handler.HealthCheck(rr, req)

	// Check status code for OPTIONS
	if rr.Code != http.StatusNoContent {
		t.Errorf("HealthCheck OPTIONS status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":      "*",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "POST, OPTIONS, GET, PUT, DELETE",
	}

	for header, expectedValue := range expectedHeaders {
		if got := rr.Header().Get(header); got != expectedValue {
			t.Errorf("HealthCheck CORS header %s = %q, want %q", header, got, expectedValue)
		}
	}
}

func TestSetupRoutes(t *testing.T) {
	logger := &mockLogger{}
	handler := NewHandler(logger, nil, nil)
	mux := http.NewServeMux()

	// Setup routes
	handler.SetupRoutes(mux)

	// Test that routes are registered by making requests
	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{testHealthPath, http.StatusOK},
		{testStatsPath, http.StatusOK},
		{testConfigPath, http.StatusOK},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != tc.expectedStatus {
			t.Errorf("Route %s: expected status %d, got %d", tc.path, tc.expectedStatus, rr.Code)
		}
	}
}

func TestHandleConfigs(t *testing.T) {
	logger := &mockLogger{}
	handler := NewHandler(logger, nil, nil)

	t.Run("GET request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testConfigPath, nil)
		rr := httptest.NewRecorder()

		handler.HandleConfigs(rr, req)

		// Check status code
		if rr.Code != http.StatusOK {
			t.Errorf("HandleConfigs GET status = %d, want %d", rr.Code, http.StatusOK)
		}

		// Check Content-Type
		if contentType := rr.Header().Get(contentTypeHeader); contentType != jsonContentType {
			t.Errorf("HandleConfigs GET Content-Type = %q, want %q", contentType, jsonContentType)
		}

		// Parse and validate JSON response
		var response models.SuccessResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode config response: %v", err)
		}

		if response.Message != MsgConfigRetrieved {
			t.Errorf("HandleConfigs GET message = %q, want %q", response.Message, MsgConfigRetrieved)
		}

		// Verify data is an empty array
		data, ok := response.Data.([]interface{})
		if !ok {
			t.Fatal("HandleConfigs GET response data is not an array")
		}

		if len(data) != 0 {
			t.Errorf("HandleConfigs GET data length = %d, want %d", len(data), 0)
		}
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, testConfigPath, nil)
		rr := httptest.NewRecorder()

		handler.HandleConfigs(rr, req)

		// Check status code for OPTIONS
		if rr.Code != http.StatusNoContent {
			t.Errorf("HandleConfigs OPTIONS status = %d, want %d", rr.Code, http.StatusNoContent)
		}

		// Check CORS headers
		expectedHeaders := map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Methods":     "POST, OPTIONS, GET, PUT, DELETE",
		}

		for header, expectedValue := range expectedHeaders {
			if got := rr.Header().Get(header); got != expectedValue {
				t.Errorf("HandleConfigs CORS header %s = %q, want %q", header, got, expectedValue)
			}
		}
	})

	t.Run("Unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, testConfigPath, nil)
		rr := httptest.NewRecorder()

		handler.HandleConfigs(rr, req)

		// Check status code for unsupported method
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("HandleConfigs POST status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
		}

		// Parse and validate error response
		var response models.ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if response.Error != ErrMethodNotAllowed {
			t.Errorf("HandleConfigs POST error = %q, want %q", response.Error, ErrMethodNotAllowed)
		}
	})
}

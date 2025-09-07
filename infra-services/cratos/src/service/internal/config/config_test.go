package config

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	configFileName           = "config.yaml"
	expectedLogLevelDebugMsg = "Expected log level 'debug', got %s"
)

func TestLoadConfigDefaults(t *testing.T) {
	config := LoadConfig()

	if config.Server.Host != "localhost" {
		t.Errorf("Expected localhost, got %s", config.Server.Host)
	}
	if config.Server.Port != 4477 {
		t.Errorf("Expected 4477, got %d", config.Server.Port)
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected info, got %s", config.Logging.Level)
	}
}

func TestLoadConfigWithEnvVars(t *testing.T) {
	// Save original values
	originalHost := os.Getenv("SERVER_HOST")
	originalPort := os.Getenv("SERVER_PORT")

	// Set test values
	os.Setenv("SERVER_HOST", "testhost")
	os.Setenv("SERVER_PORT", "9999")

	config := LoadConfig()

	if config.Server.Host != "testhost" {
		t.Errorf("Expected testhost, got %s", config.Server.Host)
	}
	if config.Server.Port != 9999 {
		t.Errorf("Expected 9999, got %d", config.Server.Port)
	}

	// Restore original values
	if originalHost != "" {
		os.Setenv("SERVER_HOST", originalHost)
	} else {
		os.Unsetenv("SERVER_HOST")
	}
	if originalPort != "" {
		os.Setenv("SERVER_PORT", originalPort)
	} else {
		os.Unsetenv("SERVER_PORT")
	}
}

func TestLoadConfigFromFileSuccess(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create a temporary config file
	configContent := `
server:
  host: "test.example.com"
  port: 9000
  read_timeout: 30
  write_timeout: 40
logging:
  level: "debug"
  format: "text"
`
	err := os.WriteFile(filepath.Join(tempDir, configFileName), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfigFromFile(filepath.Join(tempDir, configFileName))
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	// Verify the values were loaded correctly
	if config.Server.Host != "test.example.com" {
		t.Errorf("Expected host 'test.example.com', got %s", config.Server.Host)
	}
	if config.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", config.Server.Port)
	}
	if config.Logging.Level != "debug" {
		t.Errorf(expectedLogLevelDebugMsg, config.Logging.Level)
	}
}

func TestLoadConfigFromFileError(t *testing.T) {
	// Test with non-existent file
	_, err := LoadConfigFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	// Test with invalid YAML
	tempDir := t.TempDir()
	invalidYAML := `
server:
  host: "test.com
  port: invalid
database:
  - this is not valid YAML structure
`
	err = os.WriteFile(filepath.Join(tempDir, configFileName), []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = LoadConfigFromFile(filepath.Join(tempDir, configFileName))
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadConfigWithDefaultsFileExists(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `
server:
  host: "file.example.com"
  port: 7777
logging:
  level: "warn"
`

	configFile := tempDir + "/existing_config.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config := LoadConfigWithDefaults(configFile)

	// Should load from file
	if config.Server.Host != "file.example.com" {
		t.Errorf("Expected host from file 'file.example.com', got %s", config.Server.Host)
	}
	if config.Server.Port != 7777 {
		t.Errorf("Expected port from file 7777, got %d", config.Server.Port)
	}
}

func TestLoadConfigWithDefaultsFallback(t *testing.T) {
	// Test with non-existent file - should fall back to defaults
	config := LoadConfigWithDefaults("/nonexistent/config.yaml")

	// Should use default values
	if config.Server.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got %s", config.Server.Host)
	}
	if config.Server.Port != 4477 {
		t.Errorf("Expected default port 4477, got %d", config.Server.Port)
	}
}

func TestInvalidEnvVars(t *testing.T) {
	// Save original values
	originalPort := os.Getenv("SERVER_PORT")

	// Set invalid values
	os.Setenv("SERVER_PORT", "not-a-number")

	config := LoadConfig()

	// Should use default values when env vars are invalid
	if config.Server.Port != 4477 {
		t.Errorf("Expected default port 4477 for invalid env var, got %d", config.Server.Port)
	}

	// Restore original values
	if originalPort != "" {
		os.Setenv("SERVER_PORT", originalPort)
	} else {
		os.Unsetenv("SERVER_PORT")
	}
}

func TestOverrideWithEnvVars(t *testing.T) {
	// Save original values
	originalHost := os.Getenv("SERVER_HOST")
	originalLogLevel := os.Getenv("LOG_LEVEL")

	// Test override functionality
	config := &RawConfig{
		Server: RawServerConfig{
			Host: "original.com",
			Port: 4477,
		},
		Logging: RawLoggingConfig{
			Level: "info",
		},
	}

	// Set environment variables to override some values
	os.Setenv("SERVER_HOST", "override.com")
	os.Setenv("LOG_LEVEL", "debug")

	// Call the function
	overrideWithEnvVars(config)

	// Check that specified values were overridden
	if config.Server.Host != "override.com" {
		t.Errorf("Expected server host 'override.com', got %s", config.Server.Host)
	}
	if config.Logging.Level != "debug" {
		t.Errorf(expectedLogLevelDebugMsg, config.Logging.Level)
	}

	// Check that non-overridden values remained the same
	if config.Server.Port != 4477 {
		t.Errorf("Expected server port 4477 (not overridden), got %d", config.Server.Port)
	}

	// Restore original values
	if originalHost != "" {
		os.Setenv("SERVER_HOST", originalHost)
	} else {
		os.Unsetenv("SERVER_HOST")
	}
	if originalLogLevel != "" {
		os.Setenv("LOG_LEVEL", originalLogLevel)
	} else {
		os.Unsetenv("LOG_LEVEL")
	}
}

func TestOverrideWithEnvVarsAllFields(t *testing.T) {
	// Save original values
	envVars := []string{
		"SERVER_HOST", "SERVER_PORT", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT",
		"LOG_LEVEL", "LOG_FORMAT",
	}
	originalValues := make(map[string]string)
	for _, env := range envVars {
		originalValues[env] = os.Getenv(env)
	}

	// Test override functionality for all fields
	config := &RawConfig{
		Server: RawServerConfig{
			Host:         "original.com",
			Port:         4477,
			ReadTimeout:  10,
			WriteTimeout: 10,
		},
		Logging: RawLoggingConfig{
			Level: "info",
		},
	}

	// Set all environment variables
	os.Setenv("SERVER_HOST", "new.com")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_READ_TIMEOUT", "20")
	os.Setenv("SERVER_WRITE_TIMEOUT", "30")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "text")

	// Call the function
	overrideWithEnvVars(config)

	// Check all values were overridden
	if config.Server.Host != "new.com" {
		t.Errorf("Expected server host 'new.com', got %s", config.Server.Host)
	}
	if config.Server.Port != 9090 {
		t.Errorf("Expected server port 9090, got %d", config.Server.Port)
	}
	if config.Server.ReadTimeout != 20 {
		t.Errorf("Expected read timeout 20, got %d", config.Server.ReadTimeout)
	}
	if config.Server.WriteTimeout != 30 {
		t.Errorf("Expected write timeout 30, got %d", config.Server.WriteTimeout)
	}
	if config.Logging.Level != "debug" {
		t.Errorf(expectedLogLevelDebugMsg, config.Logging.Level)
	}

	// Restore original values
	for _, env := range envVars {
		if originalValues[env] != "" {
			os.Setenv(env, originalValues[env])
		} else {
			os.Unsetenv(env)
		}
	}
}

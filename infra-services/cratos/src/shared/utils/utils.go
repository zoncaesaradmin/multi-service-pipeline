package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stretchr/testify/assert/yaml"
)

// GetEnv gets an environment variable with a default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an integer environment variable with a default value
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// ResolveConfFilePath resolves the full path to config file using SERVICE_HOME/conf/ as prefix
func ResolveConfFilePath(configPath string) string {
	// If configPath is already absolute, use as-is
	if filepath.IsAbs(configPath) {
		return configPath
	}

	// For test config files (prefixed with "test_"), look in current directory first
	if strings.HasPrefix(filepath.Base(configPath), "test_") {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Get SERVICE_HOME from environment, fallback to relative path if not set
	homeDir := os.Getenv("SERVICE_HOME")
	if homeDir == "" {
		// Default to the conf directory relative to the workspace root
		homeDir = "../../../" // From src/shared/messagebus back to workspace root
	}

	// Build path: SERVICE_HOME/conf/configPath
	return filepath.Join(homeDir, "conf", configPath)
}

func LoadConfigMap(configPath string) map[string]any {
	// Read YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	// Parse YAML into a map
	config := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	return config
}

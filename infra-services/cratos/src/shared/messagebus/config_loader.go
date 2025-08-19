package messagebus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// resolveConfigPath resolves the full path to config file using SERVICE_HOME/conf/ as prefix
func resolveConfigPath(configPath string) string {
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

// LoadConsumerConfigMap loads configuration from YAML file into a map
// This map can be used directly for kafka.ConfigMap or processed for local implementation
func LoadConsumerConfigMap(configPath string) (map[string]interface{}, error) {
	return LoadConfigMap(configPath)
}

// LoadProducerConfigMap loads producer configuration from YAML file into a map
// This map can be used directly for kafka.ConfigMap or processed for local implementation
func LoadProducerConfigMap(configPath string) (map[string]interface{}, error) {
	return LoadConfigMap(configPath)
}

// LoadConfigMap loads configuration from YAML file into a map
// Generic function that can be used for both producer and consumer configs
func LoadConfigMap(configPath string) (map[string]interface{}, error) {
	// Resolve full config path using SERVICE_HOME/conf/ as prefix
	fullPath := resolveConfigPath(configPath)

	// Read YAML file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", fullPath, err)
	}

	// Parse YAML into a map
	config := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return config, nil
} // GetStringValue safely gets a string value from config map with default
func GetStringValue(config map[string]interface{}, key, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetBoolValue safely gets a bool value from config map with default
func GetBoolValue(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetIntValue safely gets an int value from config map with default
func GetIntValue(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return defaultValue
}

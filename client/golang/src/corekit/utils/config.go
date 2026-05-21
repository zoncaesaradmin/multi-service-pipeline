package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResolveConfFilePath resolves the full path to a config file using SERVICE_HOME/conf/ as prefix.
func ResolveConfFilePath(configPath string) string {
	if filepath.IsAbs(configPath) {
		return configPath
	}

	if strings.HasPrefix(filepath.Base(configPath), "test_") {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	homeDir := os.Getenv("SERVICE_HOME")
	if homeDir == "" {
		homeDir = "../../../"
	}

	return filepath.Join(homeDir, "conf", configPath)
}

// LoadConfigMap loads a YAML config file into a generic map.
// It preserves the previous behavior of returning nil on read/parse errors.
func LoadConfigMap(configPath string) map[string]any {
	config, err := LoadConfigMapWithError(configPath)
	if err != nil {
		return nil
	}
	return config
}

// LoadConfigMapWithError loads a YAML config file into a generic map and returns errors directly.
func LoadConfigMapWithError(configPath string) (map[string]any, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", configPath, err)
	}

	config := make(map[string]any)
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	return config, nil
}

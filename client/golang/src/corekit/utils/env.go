package utils

import (
	"os"
	"strconv"
	"strings"
)

// GetEnv gets an environment variable with a default value.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an integer environment variable with a default value.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool gets a boolean environment variable with a default value.
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(strings.TrimSpace(value)); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

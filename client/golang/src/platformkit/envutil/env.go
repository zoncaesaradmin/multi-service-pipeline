package envutil

import (
	"os"
	"strconv"
	"strings"
)

// Get gets an environment variable with a default value.
func Get(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetInt gets an integer environment variable with a default value.
func GetInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetBool gets a boolean environment variable with a default value.
func GetBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(strings.TrimSpace(value)); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

package messagebus

import (
	"strconv"
	"strings"
)

// GetStringValue safely gets a string value from config map with default.
func GetStringValue(config map[string]interface{}, key, defaultValue string) string {
	val, ok := config[key]
	if !ok || val == nil {
		return defaultValue
	}

	switch typed := val.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return defaultValue
	}
}

// GetBoolValue safely gets a bool value from config map with default.
func GetBoolValue(config map[string]interface{}, key string, defaultValue bool) bool {
	val, ok := config[key]
	if !ok || val == nil {
		return defaultValue
	}

	switch typed := val.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	case int:
		return typed != 0
	case int8:
		return typed != 0
	case int16:
		return typed != 0
	case int32:
		return typed != 0
	case int64:
		return typed != 0
	case uint:
		return typed != 0
	case uint8:
		return typed != 0
	case uint16:
		return typed != 0
	case uint32:
		return typed != 0
	case uint64:
		return typed != 0
	case float32:
		return typed != 0
	case float64:
		return typed != 0
	}

	return defaultValue
}

// GetIntValue safely gets an int value from config map with default.
func GetIntValue(config map[string]interface{}, key string, defaultValue int) int {
	val, ok := config[key]
	if !ok || val == nil {
		return defaultValue
	}

	switch typed := val.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}

	return defaultValue
}

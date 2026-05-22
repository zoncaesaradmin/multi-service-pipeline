package configmap

// String safely gets a string value from a config map with a default.
func String(config map[string]interface{}, key, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// Bool safely gets a bool value from a config map with a default.
func Bool(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// Int safely gets an int value from a config map with a default.
func Int(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return defaultValue
}

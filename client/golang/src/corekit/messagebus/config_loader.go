package messagebus

// GetStringValue safely gets a string value from config map with default
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

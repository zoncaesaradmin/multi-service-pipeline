package messagebus

import "corekit/internal/configmap"

// GetStringValue safely gets a string value from config map with default.
// Deprecated: use package-specific config handling; this wrapper remains for compatibility.
func GetStringValue(config map[string]interface{}, key, defaultValue string) string {
	return configmap.String(config, key, defaultValue)
}

// GetBoolValue safely gets a bool value from config map with default.
// Deprecated: use package-specific config handling; this wrapper remains for compatibility.
func GetBoolValue(config map[string]interface{}, key string, defaultValue bool) bool {
	return configmap.Bool(config, key, defaultValue)
}

// GetIntValue safely gets an int value from config map with default.
// Deprecated: use package-specific config handling; this wrapper remains for compatibility.
func GetIntValue(config map[string]interface{}, key string, defaultValue int) int {
	return configmap.Int(config, key, defaultValue)
}

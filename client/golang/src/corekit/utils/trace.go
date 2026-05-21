package utils

import (
	"context"
	"corekit/logging"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ContextKey type for trace context keys
type ContextKey string

const (
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "traceId"

	// TraceIDHeader is the header name for trace ID in Kafka messages
	TraceIDHeader = "X-Trace-Id"

	// DebugEnabledHeader is the canonical header name for enabling per-flow debug logs.
	DebugEnabledHeader = "X-Enable-Debug"
)

// GenerateTraceID creates a new random trace ID
func GenerateTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// ExtractTraceID extracts trace ID from message headers, creates one if missing
func ExtractTraceID(headers map[string]string) string {
	if traceID, exists := headers[TraceIDHeader]; exists && traceID != "" {
		return traceID
	}
	return GenerateTraceID()
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(TraceIDKey).(string)
	return traceID, ok
}

// ExtractDebugEnabled parses the external per-flow debug flag from headers.
// Missing or invalid values default to false.
func ExtractDebugEnabled(headers map[string]string) bool {
	if len(headers) == 0 {
		return false
	}

	for key, value := range headers {
		switch {
		case strings.EqualFold(key, DebugEnabledHeader):
			return parseDebugEnabled(value)
		case key == "enableDebug", key == "enable_debug", key == "debug":
			return parseDebugEnabled(value)
		}
	}
	return false
}

// WithDebugEnabled stores the per-flow debug override in the context.
func WithDebugEnabled(ctx context.Context, enabled bool) context.Context {
	return logging.WithDebugEnabled(ctx, enabled)
}

// GetDebugEnabled returns the per-flow debug override from context.
func GetDebugEnabled(ctx context.Context) (bool, bool) {
	return logging.GetDebugEnabled(ctx)
}

// BuildFlowContext creates a context carrying both trace ID and per-flow debug settings.
func BuildFlowContext(ctx context.Context, headers map[string]string) (context.Context, string) {
	traceID := ExtractTraceID(headers)
	ctx = WithTraceID(ctx, traceID)
	ctx = WithDebugEnabled(ctx, ExtractDebugEnabled(headers))
	return ctx, traceID
}

// WithTraceLogger returns a logger with trace ID field
func WithTraceLogger(logger logging.Logger, ctx context.Context) logging.Logger {
	if ctx != nil {
		logger = logger.WithContext(ctx)
	}
	if traceID, ok := GetTraceID(ctx); ok {
		return logger.WithField("traceId", traceID)
	}
	return logger
}

// EnsureTraceID ensures context has a trace ID, creates one if missing
func EnsureTraceID(ctx context.Context) (context.Context, string) {
	if traceID, ok := GetTraceID(ctx); ok {
		return ctx, traceID
	}

	traceID := GenerateTraceID()
	return WithTraceID(ctx, traceID), traceID
}

// CreateKnownTraceID creates a trace ID from scenario name
func CreateKnownTraceID(kName string) string {
	// Clean up scenario name to be a valid trace ID
	// Remove spaces and special characters, keep it readable
	traceID := ""
	for _, char := range kName {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			traceID += string(char)
		} else if char == ' ' || char == '_' || char == '-' {
			traceID += "_"
		}
	}
	return traceID
}

// CreateScenarioOutlineTraceID creates a unique trace ID for scenario outline examples
// It combines scenario name with example data to create unique identifiers
func CreateScenarioOutlineTraceID(scenarioName string, exampleData map[string]string) string {
	baseTraceID := CreateKnownTraceID(scenarioName)

	// Add example-specific suffix for uniqueness
	if len(exampleData) > 0 {
		suffix := ""
		// Create a deterministic but unique suffix from example data
		// Use key-value pairs sorted for consistency
		for key, value := range exampleData {
			cleanKey := cleanForTraceID(key)
			cleanValue := cleanForTraceID(value)
			if cleanKey != "" && cleanValue != "" {
				if suffix != "" {
					suffix += "_"
				}
				suffix += cleanKey + "_" + cleanValue
			}
		}
		if suffix != "" {
			baseTraceID += "_" + suffix
		}
	}

	return baseTraceID + "_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// cleanForTraceID cleans a string to be used in trace ID
func cleanForTraceID(input string) string {
	result := ""
	for _, char := range input {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			result += string(char)
		} else if char == ' ' || char == '_' || char == '-' || char == '.' {
			result += "_"
		}
	}
	return result
}

// CreateContextualTraceID creates appropriate trace ID based on scenario context
// Uses scenario outline trace ID if example data exists, otherwise uses simple scenario trace ID
func CreateContextualTraceID(scenarioName string, exampleData map[string]string) string {
	if len(exampleData) > 0 {
		return CreateScenarioOutlineTraceID(scenarioName, exampleData)
	}
	return CreateKnownTraceID(scenarioName)
}

// WithTraceLoggerFromID returns a logger with direct trace ID
func WithTraceLoggerFromID(logger logging.Logger, traceID string) logging.Logger {
	if traceID != "" {
		return logger.WithField("traceId", traceID)
	}
	return logger
}

func parseDebugEnabled(raw string) bool {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return false
	}

	if parsed, err := strconv.ParseBool(value); err == nil {
		return parsed
	}

	switch value {
	case "1", "y", "yes", "on", "debug":
		return true
	default:
		return false
	}
}

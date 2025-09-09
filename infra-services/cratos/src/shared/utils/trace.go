package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sharedgomodule/logging"
	"time"
)

// ContextKey type for trace context keys
type ContextKey string

const (
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "traceId"

	// TraceIDHeader is the header name for trace ID in Kafka messages
	TraceIDHeader = "X-Trace-Id"
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

// WithTraceLogger returns a logger with trace ID field
func WithTraceLogger(logger logging.Logger, ctx context.Context) logging.Logger {
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

	return baseTraceID
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

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

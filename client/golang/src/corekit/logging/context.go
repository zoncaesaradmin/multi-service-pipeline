package logging

import "context"

type contextKey string

const (
	debugEnabledContextKey contextKey = "logging.debugEnabled"
)

// WithDebugEnabled stores a per-flow debug override in the context.
// When enabled, loggers using this context honor Debug-level logs for that flow
// without changing the process-wide logger level.
func WithDebugEnabled(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, debugEnabledContextKey, enabled)
}

// GetDebugEnabled returns the per-flow debug override from the context.
func GetDebugEnabled(ctx context.Context) (bool, bool) {
	if ctx == nil {
		return false, false
	}
	enabled, ok := ctx.Value(debugEnabledContextKey).(bool)
	return enabled, ok
}

func effectiveLevel(base Level, ctx context.Context) Level {
	if enabled, ok := GetDebugEnabled(ctx); ok && enabled {
		return DebugLevel
	}
	return base
}

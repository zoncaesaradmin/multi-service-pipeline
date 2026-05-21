package utils

import (
	"context"
	"corekit/logging"
	"testing"
)

func TestExtractDebugEnabled(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{
			name:    "missing flag defaults false",
			headers: map[string]string{},
			want:    false,
		},
		{
			name: "canonical header true",
			headers: map[string]string{
				DebugEnabledHeader: "true",
			},
			want: true,
		},
		{
			name: "case insensitive header",
			headers: map[string]string{
				"x-enable-debug": "YES",
			},
			want: true,
		},
		{
			name: "legacy key false",
			headers: map[string]string{
				"enableDebug": "false",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractDebugEnabled(tt.headers); got != tt.want {
				t.Fatalf("ExtractDebugEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildFlowContext(t *testing.T) {
	headers := map[string]string{
		TraceIDHeader:      "trace-123",
		DebugEnabledHeader: "true",
	}

	ctx, traceID := BuildFlowContext(context.Background(), headers)
	if traceID != "trace-123" {
		t.Fatalf("expected trace ID trace-123, got %q", traceID)
	}

	if got, ok := GetTraceID(ctx); !ok || got != "trace-123" {
		t.Fatalf("expected trace ID in context, got %q ok=%v", got, ok)
	}

	if enabled, ok := logging.GetDebugEnabled(ctx); !ok || !enabled {
		t.Fatalf("expected debug override in context, got enabled=%v ok=%v", enabled, ok)
	}
}

func TestWithTraceLoggerAppliesTraceAndContext(t *testing.T) {
	base := logging.NewMockLogger()
	base.SetLevel(logging.InfoLevel)

	ctx := WithTraceID(context.Background(), "trace-xyz")
	ctx = WithDebugEnabled(ctx, true)

	logger := WithTraceLogger(base, ctx)
	logger.Debug("debug with flow context")

	entries := logger.(*logging.MockLogger).GetLogEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Fields["traceId"] != "trace-xyz" {
		t.Fatalf("expected traceId field trace-xyz, got %v", entry.Fields["traceId"])
	}
	if entry.Level != logging.DebugLevel {
		t.Fatalf("expected debug level log entry, got %v", entry.Level)
	}
}

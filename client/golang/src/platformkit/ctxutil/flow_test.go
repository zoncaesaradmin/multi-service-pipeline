package ctxutil

import (
	"context"
	"testing"
)

func TestExtractDebugEnabled(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{name: "missing flag defaults false", headers: map[string]string{}, want: false},
		{name: "canonical header true", headers: map[string]string{DebugEnabledHeader: "true"}, want: true},
		{name: "case insensitive header", headers: map[string]string{"x-enable-debug": "YES"}, want: true},
		{name: "legacy key false", headers: map[string]string{"enableDebug": "false"}, want: false},
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
	if enabled, ok := GetDebugEnabled(ctx); !ok || !enabled {
		t.Fatalf("expected debug override in context, got enabled=%v ok=%v", enabled, ok)
	}
}

func TestEnsureTraceID(t *testing.T) {
	ctx, traceID := EnsureTraceID(context.Background())
	if traceID == "" {
		t.Fatal("expected generated trace ID")
	}

	got, ok := GetTraceID(ctx)
	if !ok || got != traceID {
		t.Fatalf("expected context trace ID %q, got %q ok=%v", traceID, got, ok)
	}

	existingCtx := WithTraceID(context.Background(), "known-trace")
	ensuredCtx, ensuredTraceID := EnsureTraceID(existingCtx)
	if ensuredTraceID != "known-trace" {
		t.Fatalf("expected existing trace ID to be preserved, got %q", ensuredTraceID)
	}
	if got, ok := GetTraceID(ensuredCtx); !ok || got != "known-trace" {
		t.Fatalf("expected existing context trace ID, got %q ok=%v", got, ok)
	}
}

func TestCreateContextualTraceID(t *testing.T) {
	simple := CreateContextualTraceID("my scenario", nil)
	if simple != "my_scenario" {
		t.Fatalf("simple trace ID = %q, want my_scenario", simple)
	}

	outline := CreateContextualTraceID("my scenario", map[string]string{
		"fabric": "site-a",
		"kind":   "edge",
	})
	if outline == "" || outline == "my_scenario" {
		t.Fatalf("expected scenario outline trace ID to include suffix, got %q", outline)
	}
}

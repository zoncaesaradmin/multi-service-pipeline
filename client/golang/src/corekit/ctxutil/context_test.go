package ctxutil

import (
	"context"
	"testing"
)

func TestFieldsFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithRequestID(ctx, "req-456")
	ctx = WithCorrelationID(ctx, "corr-789")
	ctx = WithUserID(ctx, "user-abc")
	ctx = WithTenantID(ctx, "tenant-def")
	ctx = WithDebugEnabled(ctx, true)
	ctx = WithField(ctx, "workflowId", "wf-001")

	fields := FieldsFromContext(ctx)

	want := map[string]interface{}{
		"traceId":       "trace-123",
		"requestId":     "req-456",
		"correlationId": "corr-789",
		"userId":        "user-abc",
		"tenantId":      "tenant-def",
		"debugEnabled":  true,
		"workflowId":    "wf-001",
	}

	for key, expected := range want {
		if got := fields[key]; got != expected {
			t.Fatalf("field %q = %v, want %v", key, got, expected)
		}
	}
}

func TestFieldsFromContextLegacyKeys(t *testing.T) {
	ctx := context.WithValue(context.Background(), "traceID", "legacy-trace")
	ctx = context.WithValue(ctx, "request_id", "legacy-request")
	ctx = context.WithValue(ctx, "enableDebug", "true")

	fields := FieldsFromContext(ctx)

	if fields["traceId"] != "legacy-trace" {
		t.Fatalf("traceId = %v, want legacy-trace", fields["traceId"])
	}
	if fields["requestId"] != "legacy-request" {
		t.Fatalf("requestId = %v, want legacy-request", fields["requestId"])
	}
	if fields["debugEnabled"] != true {
		t.Fatalf("debugEnabled = %v, want true", fields["debugEnabled"])
	}
}

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
